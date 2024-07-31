package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"github.com/gorilla/mux"
	"js5-monitor/js5connection"
	"log"
	"net/http"
	"time"
)

type ServerResetInfo struct {
	LastResetTime     time.Time `json:"reset_time"`
	LastResetTimeUnix int64     `json:"last_reset_time_unix"`
	LastServerUptime  int64     `json:"last_server_uptime"`
}

var LastServerResetInfo *ServerResetInfo
var db *sql.DB

func main() {
	// Init Logger
	logFile := initLogging()
	defer logFile.Close()

	// Database
	db = initDatabase()

	// Start JS5 Monitor
	go MonitorJS5()

	// Init Router
	r := mux.NewRouter()

	// Route Handlers / Endpoints
	r.HandleFunc("/lastreset", getLastReset).Methods("GET")

	log.Fatal(http.ListenAndServe(":8081", r))
}

// -------------------------------------------------------------------------- //
// ---------------------------- Request Handlers ---------------------------- //
// -------------------------------------------------------------------------- //

func getLastReset(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	lastReset := LastServerResetInfo
	json.NewEncoder(w).Encode(lastReset)
	return
}

func consoleLoop() {
	js5Addr1 := "oldschool2.runescape.com:43594"
	js5, err := js5connection.New(js5Addr1)
	fmt.Println(err)
	//fmt.Println(js5.Ping())

	var loopCounter float64 = 0
	for {
		resp, err := js5.Ping()
		elapsedTime := js5connection.PingInterval.Seconds() * loopCounter
		fmt.Println("Resp ", resp[:10], " |  Error: ", err, " |  Seconds Elapsed: ", elapsedTime)
		if err != nil {
			fmt.Println("Connection broken! at " + time.Now().String())
			break
		}
		time.Sleep(js5connection.PingInterval)
		loopCounter++
	}

}

// MonitorJS5
//  1. Start by initializing db and getting the first server reset. If it's the first time, just set time to now.
//  2. Initialize JS5 Connection. If it fails to connect, keep trying. Server may be down.
//  3. Ping the connection every 5 seconds in a new loop.
//  4. If the connection breaks, break out of current loop
//  5. Be sure to take the difference between the time for last reset and current reset.
//  6. Write to DB
//  7. Set LastServerResetInfo to current one.
//  8. restart loop at step 2.
func MonitorJS5() {

	LastServerResetInfo = initServerResetInfo(db)

	js5AddrList := []string{
		"oldschool2.runescape.com:43594",
		"oldschool143.runescape.com:43594",
		"oldschool128.runescape.com:43594",
	}

	for { // Infinite Loop
		time.Sleep(js5connection.PingInterval)
		js5connList, err := js5connection.CreateJS5ConnectionsFromURLs(js5AddrList)
		log.Println(js5connList)
		if err != nil {
			// Start over by trying a new connection
			log.Printf("Unable to create js5connection, retrying...: %s", err.Error())
			continue
		}

		for { // Pinging Loop. Ping until connection drops
			_, err := js5connList[0].Ping()
			if err != nil {
				break
			}

			// For testing:
			//if requestKeyboardInput() {
			//	break
			//}

			time.Sleep(js5connection.PingInterval)
		}

		log.Println("JS5 Connection broken!")

		// If we broke from the loop, that means one of the connections ([0]) broke.
		// Check the rest of the connections, after the first. If any of them DON'T return an error after Ping(),
		// that means that the error may have been a false positive. Don't log the new time, and start over by
		// creating new connections.
		noErrorsOnAtLeastOneConnection := false
		for i := 1; i < len(js5connList); i++ {

			_, err := js5connList[i].Ping()
			if err == nil { // Here, we want to actually check if errors ARE nil.
				// If any ARE nil, everything is probably fine, and we restart the loop with new connections.
				noErrorsOnAtLeastOneConnection = true
				break
			}
		}
		if noErrorsOnAtLeastOneConnection {
			log.Println("Other connections were fine. Restart without logging new time")
			continue
		}

		log.Println("Logging new time")

		// If we made it here, that means all the connections were broken (all errors were not nil)
		currentServerResetInfo := ServerResetInfo{
			LastResetTime:     time.Now(),
			LastResetTimeUnix: time.Now().Unix(),
			LastServerUptime:  time.Now().Unix() - LastServerResetInfo.LastResetTimeUnix,
		}

		addNewServerResetInfo(currentServerResetInfo, db)

		LastServerResetInfo = &currentServerResetInfo
	}
}

func initServerResetInfo(database *sql.DB) *ServerResetInfo {

	var lastServerReset ServerResetInfo

	log.Println("Looking for entry in database...")
	entryExists, lastServerReset := queryDBForLastServerReset(database)

	if !entryExists { // entry does not exist in db
		lastServerReset = ServerResetInfo{
			LastResetTime:     time.Now(),
			LastResetTimeUnix: time.Now().Unix(),
			LastServerUptime:  0,
		}
		log.Println("Could not find entry in database")
		addNewServerResetInfo(lastServerReset, db)
	}

	return &lastServerReset

}

func requestKeyboardInput() bool {
	var input string
	fmt.Scanln(&input)
	return input == "y"
}
