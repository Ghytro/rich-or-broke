package tenor

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"testing"

	"github.com/Ghytro/ab_interview/config"
)

func TestGetGifById(t *testing.T) {
	gifIds := [...]string{
		"11ad486604ba6802ffe7cda95ce1f528",
		"1d73fd5b39730fd356b482128eb3746a",
		"e128f72733a6ac54534a7a47d578cfa0",
		"d04f2eaeeb55defe2a5fb19e503f0795",
		"29fc55a95c15652fe18d1422d06d7b22",
	}
	correctGifs := [len(gifIds)]*Gif{}
	testedGifs := [len(gifIds)]*Gif{}
	errs := make(chan error, len(gifIds)*2)
	var wg sync.WaitGroup
	wg.Add(len(gifIds) * 2)
	for i, id := range gifIds {
		go func(idx int, gifId string) {
			defer wg.Done()
			resp, err := http.Get(
				fmt.Sprintf(
					"https://media.tenor.com/images/%s/tenor.gif",
					gifId,
				),
			)
			if err != nil {
				errs <- err
				return
			}
			gifBinaryContent, err := io.ReadAll(resp.Body)
			if err != nil {
				errs <- err
				return
			}
			correctGifs[idx] = &Gif{gifBinaryContent}
			resp.Body.Close()
		}(i, id)
		go func(idx int, gifId string) {
			defer wg.Done()
			var err error
			testedGifs[idx], err = getGifById(gifId)
			if err != nil {
				errs <- err
				return
			}
		}(i, id)
	}
	wg.Wait()
	if len(errs) > 0 {
		t.Fatal(<-errs)
	}

	for i := range correctGifs {
		if !bytes.Equal(correctGifs[i].BinaryContent, testedGifs[i].BinaryContent) {
			t.Fatalf("Incorrect gif for id %s", gifIds[i])
		}
	}
}

type threadSafeStringSlice struct {
	S []string
	M sync.Mutex
}

func (ss *threadSafeStringSlice) Append(s string) {
	ss.M.Lock()
	defer ss.M.Unlock()
	ss.S = append(ss.S, s)
}

func (ss *threadSafeStringSlice) At(idx int) string {
	ss.M.Lock()
	defer ss.M.Unlock()
	return ss.S[idx]
}

func (ss *threadSafeStringSlice) Len() int {
	ss.M.Lock()
	defer ss.M.Unlock()
	return len(ss.S)
}

func TestGetRandomGifId(t *testing.T) {
	searchQueries := [...]string{"duck", "dog", "cat", "fish", "hello+world"}
	possibleGifs := make(map[string][]string)
	randomGifIds := make(map[string]*threadSafeStringSlice)
	errs := make(chan error, len(searchQueries)*10)
	var wg sync.WaitGroup
	wg.Add(len(searchQueries) * 2)
	for _, q := range searchQueries {
		go func(query string) {
			defer wg.Done()
			resp, err := http.Get(
				fmt.Sprintf(
					// public key here, @ShamelessLad lox
					"https://g.tenor.com/v1/search?q=%s&key=LIVDSRZULELA&limit=%d",
					query,
					config.Config.TenorSearchQueryLimit,
				),
			)
			if err != nil {
				errs <- err
				return
			}
			respBody, err := io.ReadAll(resp.Body)
			if err != nil {
				errs <- err
				return
			}
			resp.Body.Close()
			unmarshaled := new(
				struct {
					Results []struct {
						Media []struct {
							Gif struct {
								Url string `json:"url"`
							} `json:"gif"`
						} `json:"media"`
					} `json:"results"`
				},
			)
			if err := json.Unmarshal(respBody, unmarshaled); err != nil {
				errs <- err
				return
			}
			for _, r := range unmarshaled.Results {
				gifUrl := r.Media[0].Gif.Url
				splittedGifUrl := strings.Split(gifUrl, "/")
				gifId := splittedGifUrl[len(splittedGifUrl)-2]
				possibleGifs[query] = append(possibleGifs[query], gifId)
			}
		}(q)
		go func(query string) {
			defer wg.Done()
			gifId, err := getRandomGifId(query)
			if err != nil {
				errs <- err
				return
			}
			if _, ok := randomGifIds[query]; !ok {
				randomGifIds[query] = &threadSafeStringSlice{}
			}
			randomGifIds[query].Append(gifId)
		}(q)
	}
	wg.Wait()
	for query, randomGifs := range randomGifIds {
		for i := 0; i < randomGifs.Len(); i++ {
			randomGif := randomGifs.At(i)
			found := false
			for _, pg := range possibleGifs[query] {
				if pg == randomGif {
					found = true
					break
				}
			}
			if !found {
				t.Fatalf("incorrect gif id returned for search query %s: %s", query, randomGif)
			}
		}
	}
}
