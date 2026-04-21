package engine

import (
	"fmt"
	"strings"
	"sync"
	"time"

	"gorm.io/gorm/clause"

	"inscurascraper/collection/sets"
	"inscurascraper/common/number"
	"inscurascraper/common/releasename"
	"inscurascraper/engine/providerid"
	"inscurascraper/engine/rank"
	"inscurascraper/model"
	mt "inscurascraper/provider"
)

// prepareSearchKeyword normalises a user-supplied search keyword. It first
// tries release-name parsing (e.g. "Spider-Man.No.Way.Home.2021.1080p...")
// and returns the extracted clean title when PTN recognised release tags;
// when the filename carried a release year, the year is stashed onto the
// current goroutine's RequestConfig so providers with year-aware queries
// (e.g. TMDB's &year=) can pick it up.
// Otherwise it falls back to number.Trim — which is designed for JAV-style
// IDs like "ABC-123.mkv" and must NOT be applied to release names (it would
// mangle "Spider-Man.No.Way.Home..." into "Spider-Man").
func (e *Engine) prepareSearchKeyword(keyword string) string {
	if parsed := releasename.Parse(keyword); parsed.Year > 0 && parsed.Title != "" {
		if parsed.Year > 0 {
			if cfg := e.CaptureRequestConfig(); cfg != nil {
				cfg.SearchYear = parsed.Year
				e.SetRequestConfig(cfg)
			} else {
				cfg := &mt.RequestConfig{SearchYear: parsed.Year}
				e.SetRequestConfig(cfg)
			}
		}
		return parsed.Title
	}
	return number.Trim(keyword)
}

func (e *Engine) searchMovieFromDB(keyword string, provider mt.MovieProvider, all bool) (results []*model.MovieSearchResult, err error) {
	var infos []*model.MovieInfo
	tx := e.db.
		// Note: keyword might be an ID or just a regular number, so we should
		// query both of them for best match. Also, case should not matter.
		Where("number = ? COLLATE NOCASE", keyword).
		Or("id = ? COLLATE NOCASE", keyword)
	if all {
		err = tx.Find(&infos).Error
	} else {
		err = e.db.
			Where("provider = ?", provider.Name()).
			Where(tx).
			Find(&infos).Error
	}
	if err == nil {
		for _, info := range infos {
			if !info.IsValid() {
				// normally it is valid, but just in case.
				continue
			}
			results = append(results, info.ToSearchResult())
		}
	}
	return
}

func (e *Engine) searchMovie(keyword string, provider mt.MovieProvider, fallback bool) (results []*model.MovieSearchResult, err error) {
	// Regular keyword searching.
	if searcher, ok := provider.(mt.MovieSearcher); ok {
		if keyword = searcher.NormalizeMovieKeyword(keyword); keyword == "" {
			return nil, mt.ErrInvalidKeyword
		}
		if fallback {
			defer func() {
				if innerResults, innerErr := e.searchMovieFromDB(keyword, provider, false);
				// ignore DB query error.
				innerErr == nil && len(innerResults) > 0 {
					// overwrite error.
					err = nil
					// update results.
					msr := sets.NewOrderedSetWithHash(func(v *model.MovieSearchResult) string { return v.Provider + v.ID })
					msr.Add(results...)
					msr.Add(innerResults...)
					results = msr.AsSlice()
				}
			}()
		}
		return searcher.SearchMovie(keyword)
	}
	// Fallback to movie info querying.
	info, err := e.getMovieInfoByProviderID(provider, keyword, true)
	if err != nil {
		return nil, err
	}
	return []*model.MovieSearchResult{info.ToSearchResult()}, nil
}

func (e *Engine) SearchMovie(keyword, name string, fallback bool) ([]*model.MovieSearchResult, error) {
	keyword = e.prepareSearchKeyword(keyword)
	if keyword == "" {
		return nil, mt.ErrInvalidKeyword
	}
	provider, err := e.GetMovieProviderByName(name)
	if err != nil {
		return nil, err
	}
	return e.searchMovie(keyword, provider, fallback)
}

func (e *Engine) searchMovieAll(keyword string) (results []*model.MovieSearchResult, err error) {
	type response struct {
		Results   []*model.MovieSearchResult
		Error     error
		Provider  mt.MovieProvider
		StartTime time.Time
		EndTime   time.Time
	}
	respCh := make(chan response)

	// Capture per-request config before spawning goroutines.
	requestCfg := e.CaptureRequestConfig()

	var wg sync.WaitGroup
	for _, provider := range e.movieProviders.Iterator() {
		wg.Add(1)
		// Goroutine started time.
		startTime := time.Now()
		// Async searching.
		go func(provider mt.MovieProvider) {
			defer wg.Done()
			defer func() {
				if r := recover(); r != nil {
					e.logger.Printf("Panic in searchMovie for %s: %v", provider.Name(), r)
				}
			}()
			// Propagate per-request config to child goroutine.
			if requestCfg != nil {
				e.SetRequestConfigForProvider(provider, requestCfg)
				defer e.ClearRequestConfigForProvider(provider)
			}
			innerResults, innerErr := e.searchMovie(keyword, provider, false)
			respCh <- response{
				Results:   innerResults,
				Error:     innerErr,
				Provider:  provider,
				StartTime: startTime,
				EndTime:   time.Now(),
			}
		}(provider)
	}
	go func() {
		wg.Wait()
		// notify when all searching tasks done.
		close(respCh)
	}()

	ds := make([]string, 0, e.movieProviders.Len())
	// response channel.
	for resp := range respCh {
		ds = append(ds, func(a, b, c any) string {
			if c == nil {
				c = "no error"
			}
			return fmt.Sprintf("%s(%s):<%v>", a, b, c)
		}(
			resp.Provider.Name(),
			resp.EndTime.Sub(resp.StartTime),
			resp.Error,
		))

		if resp.Error != nil {
			continue
		}
		results = append(results, resp.Results...)
	}

	e.logger.Printf("Search keyword %s: %s", keyword, strings.Join(ds, " | "))
	return
}

// SearchMovieAll searches the keyword from all providers.
func (e *Engine) SearchMovieAll(keyword string, fallback bool) (results []*model.MovieSearchResult, err error) {
	keyword = e.prepareSearchKeyword(keyword)
	if keyword == "" {
		return nil, mt.ErrInvalidKeyword
	}

	defer func() {
		if err != nil {
			return
		}
		if len(results) == 0 {
			err = mt.ErrInfoNotFound
			return
		}
		// remove duplicate results, if any.
		msr := sets.NewOrderedSetWithHash(func(v *model.MovieSearchResult) string { return v.Provider + v.ID })
		msr.Add(results...)
		results = msr.AsSlice()

		// Validity filter.
		valid := make([]*model.MovieSearchResult, 0, len(results))
		for _, r := range results {
			if !r.IsValid() {
				continue
			}
			if _, err := e.GetMovieProviderByName(r.Provider); err != nil {
				e.logger.Printf("ignore provider %s as not found", r.Provider)
				continue
			}
			valid = append(valid, r)
		}

		// Apply tiered ranking + per-provider round-robin promotion.
		cfg := e.CaptureRequestConfig()
		results = rank.RankMovies(valid, rank.MovieCriteria{
			Keyword:     keyword,
			Year:        cfg.SearchYearOr(0),
			Language:    cfg.LanguageOr(""),
			MaxPriority: e.maxMovieProviderPriority(),
			PriorityOf:  e.movieProviderPriority,
		})
	}()

	if fallback /* query database for missing results  */ {
		defer func() {
			if innerResults, innerErr := e.searchMovieFromDB(keyword, nil, true);
			// ignore DB query error.
			innerErr == nil && len(innerResults) > 0 {
				// overwrite error.
				err = nil
				// append results.
				results = append(results, innerResults...)
			}
		}()
	}

	results, err = e.searchMovieAll(keyword)
	return
}

func (e *Engine) getMovieInfoFromDB(provider mt.MovieProvider, id string) (*model.MovieInfo, error) {
	info := &model.MovieInfo{}
	err := e.db. // Exact match here.
			Where("provider = ?", provider.Name()).
			Where("id = ? COLLATE NOCASE", id).
			First(info).Error
	return info, err
}

func (e *Engine) getMovieInfoWithCallback(provider mt.MovieProvider, id string, lazy bool, callback func() (*model.MovieInfo, error)) (info *model.MovieInfo, err error) {
	defer func() {
		// metadata validation check.
		if err == nil && (info == nil || !info.IsValid()) {
			err = mt.ErrIncompleteMetadata
		}
	}()
	// Query DB first (by id).
	if lazy {
		if info, err = e.getMovieInfoFromDB(provider, id); err == nil && info.IsValid() {
			return // ignore DB query error.
		}
	}
	// delayed info auto-save.
	defer func() {
		if err == nil && info.IsValid() {
			e.db.Clauses(clause.OnConflict{
				UpdateAll: true,
			}).Create(info) // ignore error
		}
	}()
	return callback()
}

func (e *Engine) getMovieInfoByProviderID(provider mt.MovieProvider, id string, lazy bool) (*model.MovieInfo, error) {
	if id = provider.NormalizeMovieID(id); id == "" {
		return nil, mt.ErrInvalidID
	}
	return e.getMovieInfoWithCallback(provider, id, lazy, func() (*model.MovieInfo, error) {
		return provider.GetMovieInfoByID(id)
	})
}

func (e *Engine) GetMovieInfoByProviderID(pid providerid.ProviderID, lazy bool) (*model.MovieInfo, error) {
	provider, err := e.GetMovieProviderByName(pid.Provider)
	if err != nil {
		return nil, err
	}
	return e.getMovieInfoByProviderID(provider, pid.ID, lazy)
}

func (e *Engine) getMovieInfoByProviderURL(provider mt.MovieProvider, rawURL string, lazy bool) (*model.MovieInfo, error) {
	id, err := provider.ParseMovieIDFromURL(rawURL)
	switch {
	case err != nil:
		return nil, err
	case id == "":
		return nil, mt.ErrInvalidURL
	}
	return e.getMovieInfoWithCallback(provider, id, lazy, func() (*model.MovieInfo, error) {
		return provider.GetMovieInfoByURL(rawURL)
	})
}

func (e *Engine) GetMovieInfoByURL(rawURL string, lazy bool) (*model.MovieInfo, error) {
	provider, err := e.GetMovieProviderByURL(rawURL)
	if err != nil {
		return nil, err
	}
	return e.getMovieInfoByProviderURL(provider, rawURL, lazy)
}
