package main

import (
	"fmt"
	"github.com/PuerkitoBio/goquery"
	"gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
	"net/url"
	"os"
	"strconv"
	"strings"
)

type Track struct {
	Id       bson.ObjectId `json:"_id" bson:"_id"`
	Title    string        `json:"title"  bson:"title"`
	Artist   string        `json:"artist" bson:"artists"`
	Link     string        `json:"link" bson:"link"`
	Quality  string        `json:"quality" bson:"quality"`
	Duration int           `json:"duration" bson:"duration"`
	Sources  []Source      `json:"sources" bson:"sources"`
}

type Source struct {
	Quality string `json:"quality"`
	Source  string `json:"source"`
}

var (
	tracks *mgo.Collection
)

func main() {
	session, err := mgo.Dial(os.Getenv("MONGODB_HOST"))
	if err != nil {
		panic(err)
	}
	defer session.Close()

	db := session.DB("csndb")
	tracks = db.C("tracks")
	//tracks.RemoveAll(nil)
	for _, artist := range []string{} {
		page := 1
		for {
			if crawByArtist(artist, page) {
				page++
			} else {
				break
			}
		}
	}

	CrawSources()

	fmt.Println("Done")

	//http: //search.chiasenhac.vn/search.php?mode=artist&s=Aerosmith&order=quality&cat=music&page=2
}

func CrawSources() {
	allTracks := []Track{}
	tracks.Find(nil).All(&allTracks)
	for idx, t := range allTracks {
		fmt.Printf("%5d %s\n", idx, t.Id.Hex())
		downloadUrl := strings.Replace(t.Link, ".html", "_download.html", -1)
		doc, _ := goquery.NewDocument(downloadUrl)
		sources := []Source{}
		doc.Find("#downloadlink2 a").Each(func(__ int, link *goquery.Selection) {
			if link.Find("span").Length() > 0 {
				source := Source{
					Source:  link.AttrOr("href", ""),
					Quality: link.Find("span").First().Text(),
				}
				sources = append(sources, source)
			}
		})
		t.Sources = sources
		tracks.UpdateId(t.Id, t)
	}
}

func crawByArtist(a string, p int) bool {
	raw := fmt.Sprintf("http://search.chiasenhac.vn/search.php?mode=artist&s=%s&order=quality&cat=music&page=%d", url.QueryEscape(a), p)
	fmt.Println("Querying " + raw)
	doc, err := goquery.NewDocument(raw)
	if err != nil {
		panic(err)
	}
	trackOnPage := 0
	doc.Find(".page-dsms tbody tr").EachWithBreak(func(idx int, s *goquery.Selection) bool {
		if idx == 0 {
			return true
		}
		newTrack := Track{
			Id: bson.NewObjectId(),
		}
		s.Find("td").Each(func(col int, r *goquery.Selection) {
			if col == 1 {
				r.Find("p").Each(func(idx int, p *goquery.Selection) {
					if idx == 0 {
						anchor := p.Find("a").First()
						newTrack.Title = anchor.Text()
						newTrack.Link = anchor.AttrOr("href", "")
					} else if idx == 1 {
						newTrack.Artist = p.Text()
					}
				})
			} else if col == 2 {
				r.Find("span").Each(func(iii int, span *goquery.Selection) {
					if iii == 1 {
						newTrack.Quality = span.Text()
					}
				})

				str := r.Find("span").First().Text()
				newTrack.Duration = GetSecondFromString(strings.Replace(str, newTrack.Quality, "", -1))
			}
		})

		if newTrack.Quality != "Lossless" {
			return false
		}

		tracks.Insert(newTrack)
		trackOnPage++
		return true
	})

	return trackOnPage >= 25
}

func GetSecondFromString(input string) int {
	chunks := strings.Split(input, ":")
	min, _ := strconv.Atoi(chunks[0])
	sec, _ := strconv.Atoi(chunks[1])
	return min*60 + sec
}
