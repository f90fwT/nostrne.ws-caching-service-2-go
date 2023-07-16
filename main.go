package main

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"os"

	"github.com/jmoiron/sqlx"
	_ "modernc.org/sqlite"

	"github.com/nbd-wtf/go-nostr"
	decodepay "github.com/nbd-wtf/ln-decodepay"
)

var db *sqlx.DB

func initDb() {
	path := ".data"
	if _, err := os.Stat(path); errors.Is(err, os.ErrNotExist) {
		err := os.Mkdir(path, os.ModePerm)
		if err != nil {
			log.Println(err)
		}
	}

	var err error
	db, err = sqlx.Open("sqlite", "./.data/sqlite.db")
	if err != nil {
		log.Fatal(err)
	}

	db.SetMaxOpenConns(1)

	createTableQuery := `
		PRAGMA journal_mode=WAL;
		CREATE TABLE IF NOT EXISTS notes (
			ID TEXT,
			Sig TEXT,
			PubKey TEXT,
			CreatedAt TEXT,
			Kind TEXT,
			Tags TEXT,
			Content TEXT,
			Upvotes INT,
			Zaps INT
		);
		CREATE TABLE IF NOT EXISTS reactions (
			ID TEXT,
			Sig TEXT,
			PubKey TEXT,
			CreatedAt TEXT,
			Kind TEXT,
			Tags TEXT,
			Content TEXT
		);
		CREATE TABLE IF NOT EXISTS zaps (
			ID TEXT,
			Sig TEXT,
			PubKey TEXT,
			CreatedAt TEXT,
			Kind TEXT,
			Tags TEXT,
			Content TEXT,
			Amount TEXT
		);
	`

	_, err = db.Exec(createTableQuery)
	if err != nil {
		log.Fatal(err)
	}
}

func main() {
	initDb()
	createConnection("wss://nos.lol")
}

/*
   UPDATE notes
   SET Upvotes = (
     SELECT COUNT(*)
     FROM reactions
     WHERE reactions.Tags LIKE '%e","' || notes.ID || '%' AND reactions.Content = '+'
   )
*/

/*
UPDATE notes
SET Upvotes = (
  SELECT COUNT(*)
  FROM reactions
  WHERE reactions.Tags LIKE '%e","' || notes.ID || '%' AND reactions.Content = '+'
)
*/

func calculateReactions() {
	// _, err := db.Exec(`
	// UPDATE notes SET Upvotes = (SELECT COUNT(*) FROM reactions WHERE reactions.Tags LIKE '%e","' || notes.ID || '%' AND reactions.Content = '+');
	// UPDATE notes SET Zaps = COALESCE((SELECT SUM(Amount) FROM zaps WHERE zaps.Tags LIKE '%e","' || notes.ID || '%'), 0);
	// `)
	// if err != nil {
	// 	log.Fatal(err)
	// }
}

func createConnection(target string) {
	ctx := context.Background()

	relay, err := nostr.RelayConnect(ctx, target)
	if err != nil {
		panic(err)
	}

	var filters = []nostr.Filter{{
		Kinds: []int{nostr.KindTextNote, nostr.KindReaction, nostr.KindZap},
		Limit: 1,
	}}

	sub, err := relay.Subscribe(ctx, filters)
	if err != nil {
		panic(err)
	}

	for ev := range sub.Events {
		if ev.Kind == nostr.KindTextNote {
			go func() {
				stmt, err := db.Prepare("INSERT INTO notes(ID, Sig, PubKey, CreatedAt, Kind, Tags, Content, Upvotes, Zaps) VALUES(?, ?, ?, ?, ?, ?, ?, ?, ?)")
				if err != nil {
					log.Printf("Error while preparing database statement: %s", err.Error())
					return
				}
				defer stmt.Close()

				jsonTags, err := json.Marshal(ev.Tags)

				_, err = stmt.Exec(ev.ID, ev.Sig, ev.PubKey, ev.CreatedAt, ev.Kind, jsonTags, ev.Content, 0, 0)
				if err != nil {
					log.Printf("Error while inserting notes into the database: %s", err.Error())
					return
				}

				calculateReactions()
			}()
		} else if ev.Kind == nostr.KindReaction {
			go func() {
				stmt, err := db.Prepare("INSERT INTO reactions(ID, Sig, PubKey, CreatedAt, Kind, Tags, Content) VALUES(?, ?, ?, ?, ?, ?, ?)")
				if err != nil {
					log.Printf("Error while preparing database statement: %s", err.Error())
					return
				}
				defer stmt.Close()

				jsonTags, err := json.Marshal(ev.Tags)

				_, err = stmt.Exec(ev.ID, ev.Sig, ev.PubKey, ev.CreatedAt, ev.Kind, jsonTags, ev.Content)
				if err != nil {
					log.Printf("Error while inserting reactions into the database: %s", err.Error())
					return
				}

				calculateReactions()
			}()
		} else if ev.Kind == nostr.KindZap {
			go func() {
				stmt, err := db.Prepare("INSERT INTO zaps(ID, Sig, PubKey, CreatedAt, Kind, Tags, Content, Amount) VALUES(?, ?, ?, ?, ?, ?, ?, ?)")
				if err != nil {
					log.Printf("Error while preparing database statement: %s", err.Error())
					return
				}
				defer stmt.Close()

				jsonTags, err := json.Marshal(ev.Tags)

				var filter = []string{
					"bolt11",
				}

				invoice := ev.Tags.GetAll(filter)

				if len(invoice) > 0 && len(invoice[0]) > 1 {
				} else {
					log.Println("zapAmount[0][1] doesn't exist")
					return
				}

				bolt11, err := decodepay.Decodepay(invoice[0][1])
				if err != nil {
					log.Println("Error while decoding invoice")
					return
				}

				_, err = stmt.Exec(ev.ID, ev.Sig, ev.PubKey, ev.CreatedAt, ev.Kind, jsonTags, ev.Content, bolt11.MSatoshi/1000)
				if err != nil {
					log.Printf("Error while inserting zaps into the database: %s", err.Error())
					return
				}

				calculateReactions()
			}()
		}
	}
}

//

// func createConnection0(target string) {
// 	ctx := context.Background()
// 	relay, err := nostr.RelayConnect(ctx, target)
// 	if err != nil {
// 		panic(err)
// 	}

// 	var filters = []nostr.Filter{{
// 		Kinds: []int{nostr.KindTextNote},
// 		Limit: 1,
// 	}}

// 	sub, err := relay.Subscribe(ctx, filters)
// 	if err != nil {
// 		panic(err)
// 	}

// 	for ev := range sub.Events {
// 		//for _, tag := range ev.Tags {
// 		//if len(tag) == 2 && tag[0] == "t" && tag[1] == "plebchain" {
// 		go func() {
// 			stmt, err := db.Prepare("INSERT INTO notes(ID, Sig, PubKey, CreatedAt, Kind, Tags, Content, Upvotes) VALUES(?, ?, ?, ?, ?, ?, ?, ?)")
// 			if err != nil {
// 				log.Printf("Error while preparing database statement: %s", err.Error())
// 				return
// 			}
// 			defer stmt.Close()

// 			jsonTags, err := json.Marshal(ev.Tags)

// 			_, err = stmt.Exec(ev.ID, ev.Sig, ev.PubKey, ev.CreatedAt, ev.Kind, jsonTags, ev.Content, 0)
// 			if err != nil {
// 				log.Printf("Error while inserting notes into the database: %s", err.Error())
// 				return
// 			}
// 		}()
// 		//break
// 		//}
// 		//}
// 	}
// 	calculateLikes()
// }

// func createConnectionKind7(target string) {
// 	ctx := context.Background()
// 	relay, err := nostr.RelayConnect(ctx, target)
// 	if err != nil {
// 		panic(err)
// 	}

// 	var filters = []nostr.Filter{{
// 		Kinds: []int{nostr.KindReaction},
// 		Limit: 1,
// 	}}

// 	sub, err := relay.Subscribe(ctx, filters)
// 	if err != nil {
// 		panic(err)
// 	}

// 	for ev := range sub.Events {
// 		go func() {
// 			stmt, err := db.Prepare("INSERT INTO reactions(ID, Sig, PubKey, CreatedAt, Kind, Tags, Content) VALUES(?, ?, ?, ?, ?, ?, ?)")
// 			if err != nil {
// 				log.Printf("Error while preparing database statement: %s", err.Error())
// 				return
// 			}
// 			defer stmt.Close()

// 			jsonTags, err := json.Marshal(ev.Tags)

// 			_, err = stmt.Exec(ev.ID, ev.Sig, ev.PubKey, ev.CreatedAt, ev.Kind, jsonTags, ev.Content)
// 			if err != nil {
// 				log.Printf("Error while inserting reactions into the database: %s", err.Error())
// 				return
// 			}
// 		}()
// 		calculateLikes()
// 	}
// }
