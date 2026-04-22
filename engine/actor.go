package engine

import (
	goerr "errors"
	"fmt"
	"inscurascraper/collection/sets"
	"inscurascraper/collection/slices"
	"inscurascraper/common/comparer"
	"inscurascraper/common/parser"
	"inscurascraper/engine/providerid"
	"inscurascraper/engine/rank"
	"inscurascraper/model"
	"sort"
	"sync"

	mt "inscurascraper/provider"

	"gorm.io/gorm/clause"
)

func (e *Engine) searchActorFromDB(keyword string, provider mt.Provider) (results []*model.ActorSearchResult, err error) {
	var infos []*model.ActorInfo
	if err = e.db.
		Where("provider = ? AND name = ? COLLATE NOCASE",
			provider.Name(), keyword).
		Find(&infos).Error; err == nil {
		for _, info := range infos {
			if !info.IsValid() {
				continue
			}
			results = append(results, info.ToSearchResult())
		}
	}
	return
}

func (e *Engine) searchActor(keyword string, provider mt.Provider, fallback bool) ([]*model.ActorSearchResult, error) {
	innerSearch := func(keyword string) (results []*model.ActorSearchResult, err error) {
		if searcher, ok := provider.(mt.ActorSearcher); ok {
			defer func() {
				if err != nil || len(results) == 0 {
					return // ignore error or empty.
				}
				const minSimilarity = 0.3
				ps := new(slices.WeightedSlice[*model.ActorSearchResult, float64])
				for _, result := range results {
					if similarity := comparer.Compare(result.Name, keyword); similarity >= minSimilarity {
						ps.Append(result, similarity)
					}
				}
				results = ps.SortFunc(sort.Stable).Slice() // replace results.
			}()
			if fallback {
				defer func() {
					if innerResults, innerErr := e.searchActorFromDB(keyword, provider);
					// ignore DB query error.
					innerErr == nil && len(innerResults) > 0 {
						// overwrite error.
						err = nil
						// update results.
						asr := sets.NewOrderedSetWithHash(func(v *model.ActorSearchResult) string { return v.Provider + v.ID })
						// unlike movie searching, we want search results go first
						// than DB data here, so we add results later than DB results.
						asr.Add(innerResults...)
						asr.Add(results...)
						results = asr.AsSlice()
					}
				}()
			}
			return searcher.SearchActor(keyword)
		}
		// All providers should implement the ActorSearcher interface.
		return nil, mt.ErrInfoNotFound
	}
	names := parser.ParseActorNames(keyword)
	if len(names) == 0 {
		return nil, mt.ErrInvalidKeyword
	}
	var (
		results []*model.ActorSearchResult
		errors  []error
	)
	for _, name := range names {
		innerResults, innerErr := innerSearch(name)
		if innerErr != nil &&
			// ignore InfoNotFound error.
			!goerr.Is(innerErr, mt.ErrInfoNotFound) {
			// add error to chain and handle it later.
			errors = append(errors, innerErr)
			continue
		}
		results = append(results, innerResults...)
	}
	if len(results) == 0 {
		if len(errors) > 0 {
			return nil, fmt.Errorf("search errors: %v", errors)
		}
		return nil, mt.ErrInfoNotFound
	}
	return results, nil
}

func (e *Engine) SearchActor(keyword, name string, fallback bool) ([]*model.ActorSearchResult, error) {
	provider, err := e.GetActorProviderByName(name)
	if err != nil {
		return nil, err
	}
	return e.searchActor(keyword, provider, fallback)
}

func (e *Engine) SearchActorAll(keyword string, fallback bool) (results []*model.ActorSearchResult, err error) {
	var (
		mu sync.Mutex
		wg sync.WaitGroup
	)
	// Capture per-request config before spawning goroutines.
	requestCfg := e.CaptureRequestConfig()

	for _, provider := range e.actorProviders.Iterator() {
		wg.Add(1)
		go func(provider mt.ActorProvider) {
			defer wg.Done()
			defer func() {
				if r := recover(); r != nil {
					e.logger.Printf("Panic in searchActor for %s: %v", provider.Name(), r)
				}
			}()
			// Propagate per-request config to child goroutine.
			if requestCfg != nil {
				e.SetRequestConfigForProvider(provider, requestCfg)
				defer e.ClearRequestConfigForProvider(provider)
			}
			if innerResults, innerErr := e.searchActor(keyword, provider, fallback); innerErr == nil {
				for _, result := range innerResults {
					if result.IsValid() /* validation check */ {
						mu.Lock()
						results = append(results, result)
						mu.Unlock()
					}
				}
			} // ignore error
		}(provider)
	}
	wg.Wait()

	// Apply tiered ranking + per-provider round-robin promotion.
	cfg := e.CaptureRequestConfig()
	results = rank.RankActors(results, rank.ActorCriteria{
		Keyword:     keyword,
		Language:    cfg.LanguageOr(""),
		MaxPriority: e.maxActorProviderPriority(),
		PriorityOf:  e.actorProviderPriority,
	})
	return
}

func (e *Engine) getActorInfoFromDB(provider mt.ActorProvider, id string) (*model.ActorInfo, error) {
	info := &model.ActorInfo{}
	err := e.db. // Exact match here.
			Where("provider = ?", provider.Name()).
			Where("id = ? COLLATE NOCASE", id).
			First(info).Error
	return info, err
}

func (e *Engine) getActorInfoWithCallback(provider mt.ActorProvider, id string, lazy bool, callback func() (*model.ActorInfo, error)) (info *model.ActorInfo, err error) {
	defer func() {
		// metadata validation check.
		if err == nil && (info == nil || !info.IsValid()) {
			err = mt.ErrIncompleteMetadata
		}
	}()
	// Query DB first (by id).
	if lazy {
		if info, err = e.getActorInfoFromDB(provider, id); err == nil && info.IsValid() {
			return
		}
	}
	// Delayed info auto-save.
	defer func() {
		if err == nil && info.IsValid() {
			// Make sure we save the original info here.
			e.db.Clauses(clause.OnConflict{
				UpdateAll: true,
			}).Create(info) // ignore error
		}
	}()
	return callback()
}

func (e *Engine) getActorInfoByProviderID(provider mt.ActorProvider, id string, lazy bool) (*model.ActorInfo, error) {
	if id = provider.NormalizeActorID(id); id == "" {
		return nil, mt.ErrInvalidID
	}
	return e.getActorInfoWithCallback(provider, id, lazy, func() (*model.ActorInfo, error) {
		return provider.GetActorInfoByID(id)
	})
}

func (e *Engine) GetActorInfoByProviderID(pid providerid.ProviderID, lazy bool) (*model.ActorInfo, error) {
	provider, err := e.GetActorProviderByName(pid.Provider)
	if err != nil {
		return nil, err
	}
	return e.getActorInfoByProviderID(provider, pid.ID, lazy)
}

func (e *Engine) getActorInfoByProviderURL(provider mt.ActorProvider, rawURL string, lazy bool) (*model.ActorInfo, error) {
	id, err := provider.ParseActorIDFromURL(rawURL)
	switch {
	case err != nil:
		return nil, err
	case id == "":
		return nil, mt.ErrInvalidURL
	}
	return e.getActorInfoWithCallback(provider, id, lazy, func() (*model.ActorInfo, error) {
		return provider.GetActorInfoByURL(rawURL)
	})
}

func (e *Engine) GetActorInfoByURL(rawURL string, lazy bool) (*model.ActorInfo, error) {
	provider, err := e.GetActorProviderByURL(rawURL)
	if err != nil {
		return nil, err
	}
	return e.getActorInfoByProviderURL(provider, rawURL, lazy)
}
