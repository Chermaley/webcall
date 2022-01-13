// WebCall Copyright 2021 timur.mobi. All rights reserved.
//
// httpLogin() is called by callees via XHR "/rtcsig/login". 
// httpLogin() makes sure that the given urlID and password 
// (or the stored cookie) are the same as during registration.
// Cookie support is not required for a successful login.
// If cookies are supported by the client, a cookie is stored
// to allow for convenient reconnect. On successful login, the 
// callee client will receive a responseString in the form of 
// "wss://(hostname):(wssPort)/ws|other|parameters|...|..."
// with which the websocket connection can be established.

package main

import (
	"net/http"
	"time"
	"strings"
	"fmt"
	"io"
	"os"
	"math/rand"
	"sync"
)

func httpLogin(w http.ResponseWriter, r *http.Request, urlID string, cookie *http.Cookie, pw string, remoteAddr string, remoteAddrWithPort string, nocookie bool, startRequestTime time.Time, pwIdCombo PwIdCombo, userAgent string) {
	//fmt.Printf("/login urlID=(%s) rip=%s rt=%v\n",
	//	urlID, remoteAddrWithPort, time.Since(startRequestTime)) // rt=4.393µs

	// reached maxCallees?
	hubMapMutex.RLock()
	lenHubMap := len(hubMap)
	hubMapMutex.RUnlock()
	readConfigLock.RLock()
	myMaxCallees := maxCallees
	readConfigLock.RUnlock()
	if lenHubMap > myMaxCallees {
		fmt.Printf("# /login lenHubMap %d > myMaxCallees %d rip=%s\n", lenHubMap, myMaxCallees, remoteAddr)
		fmt.Fprintf(w, "error")
		return
	}

	readConfigLock.RLock()
	myMultiCallees := multiCallees
	readConfigLock.RUnlock()

	if strings.Index(myMultiCallees, "|"+urlID+"|") < 0 {
		// urlID is NOT a multiCallee user
		// so if urlID is already logged-in, we must abort
		// unless the request comes from the same IP, in which case we log the old session out
// TODO however, we must also disconnect the old session so that the client disconnects
//      and does not try to login again
		ejectOn1stFound := true
		reportHiddenCallee := true
		occupy := false
		key, _, _, err := GetOnlineCallee(urlID, ejectOn1stFound, reportHiddenCallee,
			remoteAddr, occupy, "/login")
		if err != nil {
			fmt.Printf("# /login key=(%s) GetOnlineCallee() err=%v\n", key, err)
		}
		if key != "" {
			// found "already logged in"
			// delay a bit to see if we receive a parallel exithub that might delete this key
			time.Sleep(1000 * time.Millisecond)
			// check again
			key, _, _, err = GetOnlineCallee(urlID, ejectOn1stFound, reportHiddenCallee,
				remoteAddr, occupy, "/login")
			if err != nil {
				fmt.Printf("# /login key=(%s) GetOnlineCallee() err=%v\n", key, err)
			}
			if key != "" {
				// "already logged in" entry still exists
				// if remoteAddr == hub.CalleeClient.RemoteAddrNoPort: unregister old entry
				hubMapMutex.RLock()
				hub := hubMap[key]
				hubMapMutex.RUnlock()
				offlineReason := 0
				if hub==nil {
					offlineReason = 1 // callee's hub is gone
				} else if hub.CalleeClient==nil {
					offlineReason = 2 // CalleeClient is gone
				} else if !hub.CalleeClient.isOnline.Get() {
					offlineReason = 3 // CalleeClient is not online anymore
				} else {
					// hub.CalleeClient seems to be online; let's see if this holds if we ping it
					fmt.Printf("/login key=(%s) send ping to prev rip=%s\n", key, remoteAddr)
					hub.CalleeClient.SendPing(2000)
					time.Sleep(2200 * time.Millisecond)
					// is hub.CalleeClient still online now?
					if hub==nil || hub.CalleeClient==nil || !hub.CalleeClient.isOnline.Get() {
						offlineReason = 4 // CalleeClient is not online anymore
					}
				}
				if offlineReason==0 {
					// abort login: old callee is still online
					fmt.Fprintf(w, "fatal")
					fmt.Printf("/login key=(%s) is already logged in (%d) rip=%s ua=%s\n",
						key, offlineReason, remoteAddr, userAgent)
					return
				}
				// apparenly the new login comes from the old callee, bc it is not online anymore
				// no need to hub.doUnregister(hub.CalleeClient, ""); just continue with the login
			}
		}
	}

	//fmt.Printf("/login pw before httpPost (%s)\n", pw)
	postBuf := make([]byte, 128)
	length, _ := io.ReadFull(r.Body, postBuf)
	if length > 0 {
		var pwData = string(postBuf[:length])
		//fmt.Printf("/login pwData (%s)\n", pwData)
		pwData = strings.ToLower(pwData)
		pwData = strings.TrimSpace(pwData)
		tokenSlice := strings.Split(pwData, "&")
		for _, tok := range tokenSlice {
			if strings.HasPrefix(tok, "pw=") {
				pwFromPost := tok[3:]
				if(pwFromPost!="") {
					pw = pwFromPost
					//fmt.Printf("/login pw from httpPost (%s)\n", pw)
					break
				}
			}
		}
	}

	// pw must be available now
	if pw == "" {
		fmt.Printf("/login no pw urlID=%s rip=%s ua=%s\n", urlID, remoteAddr, r.UserAgent())
		fmt.Fprintf(w, "error")
		return
	}

	//fmt.Printf("/login pw given urlID=(%s) rip=%s rt=%v\n",
	//	urlID, remoteAddr, time.Since(startRequestTime)) // rt=23.184µs
	var dbEntry DbEntry
	var dbUser DbUser
	var wsClientID uint64
	var lenGlobalHubMap int64
	serviceSecs := 0
	globalID := ""

	if strings.HasPrefix(urlID, "random") {
		// ignore
	} else if strings.HasPrefix(urlID, "!") {
		// create new unique wsClientID
		wsClientMutex.Lock()
		wsClientID = getNewWsClientID()
		wsClientMutex.Unlock()
		//fmt.Printf("/login set wsClientMap[%d] for ID=(%s)\n", wsClientID, globalID)
		// hub.WsClientID and hub.ConnectedCallerIp will be set by wsclient.go

		var err error
		globalID,_,err = StoreCalleeInHubMap(urlID, myMultiCallees, remoteAddrWithPort, wsClientID, false)
		if err != nil || globalID == "" {
			fmt.Printf("# /login id=(%s) StoreCalleeInHubMap(%s) err=%v\n", globalID, urlID, err)
			fmt.Fprintf(w, "noservice")
			return
		}
		//fmt.Printf("/login globalID=(%s) urlID=(%s) rip=%s rt=%v\n",
		//	globalID, urlID, remoteAddr, time.Since(startRequestTime))
	} else {
		// pw check for everyone other than random and duo
		if len(pw) < 6 {
			// guessing more difficult if delayed
			fmt.Printf("/login pw too short urlID=(%s) rip=%s\n", urlID, remoteAddr)
			time.Sleep(3000 * time.Millisecond)
			fmt.Fprintf(w, "error")
			return
		}

		err := kvMain.Get(dbRegisteredIDs, urlID, &dbEntry)
		if err != nil {
			fmt.Printf("# /login error db=%s bucket=%s key=%s get registeredID err=%v\n",
				dbMainName, dbRegisteredIDs, urlID, err)
			if strings.Index(err.Error(), "disconnect") >= 0 {
				// TODO admin email notif may be useful
				fmt.Fprintf(w, "error")
				return
			}
			if strings.Index(err.Error(), "timeout") < 0 {
				// guessing more difficult if delayed
				time.Sleep(3000 * time.Millisecond)
			}
			fmt.Fprintf(w, "notregistered")
			return
		}

		if pw != dbEntry.Password {
			fmt.Fprintf(os.Stderr, "/login fail id=%s wrong password rip=%s\n", urlID, remoteAddr)
			// must delay to make guessing more difficult
			time.Sleep(3000 * time.Millisecond)
			fmt.Fprintf(w, "error")
			return
		}

		// pw accepted
		dbUserKey := fmt.Sprintf("%s_%d", urlID, dbEntry.StartTime)
		err = kvMain.Get(dbUserBucket, dbUserKey, &dbUser)
		if err != nil {
			fmt.Printf("# /login error db=%s bucket=%s get key=%v err=%v\n",
				dbMainName, dbUserBucket, dbUserKey, err)
			fmt.Fprintf(w, "error")
			return
		}
		//fmt.Printf("/login dbUserKey=%v dbUser.Int=%d (hidden) rt=%v\n",
		//	dbUserKey, dbUser.Int2, time.Since(startRequestTime)) // rt=75ms

		// store dbUser with modified LastLoginTime
		dbUser.LastLoginTime = time.Now().Unix()
		err = kvMain.Put(dbUserBucket, dbUserKey, dbUser, false)
		if err!=nil {
			fmt.Printf("# /login error db=%s bucket=%s put key=%s err=%v\n",
				dbMainName, dbUserBucket, urlID, err)
		}

		// create new unique wsClientID
		wsClientMutex.Lock()
		wsClientID = getNewWsClientID()
		wsClientMutex.Unlock()
		//fmt.Printf("/login set wsClientMap[%d] for ID=(%s)\n", wsClientID, globalID)
		// hub.WsClientID and hub.ConnectedCallerIp will be set by wsclient.go

		globalID,_,err = StoreCalleeInHubMap(urlID, myMultiCallees, remoteAddrWithPort, wsClientID, false)
		if err != nil || globalID == "" {
			fmt.Printf("# /login id=(%s) StoreCalleeInHubMap(%s) err=%v\n", globalID, urlID, err)
			fmt.Fprintf(w, "noservice")
			return
		}
		//fmt.Printf("/login globalID=(%s) urlID=(%s) rip=%s rt=%v\n",
		//	globalID, urlID, remoteAddr, time.Since(startRequestTime))

		if cookie == nil && !nocookie {
			err,cookieValue := createCookie(w, urlID, pw, &pwIdCombo)
			if err != nil {
				fmt.Printf("# /login persist PwIdCombo error db=%s bucket=%s cookie=%s err=%v\n",
					dbHashedPwName, dbHashedPwBucket, cookieValue, err)
				if globalID != "" {
					_,lenGlobalHubMap = DeleteFromHubMap(globalID)
				}
				fmt.Fprintf(w, "noservice")
				return
			}

			if logWantedFor("cookie") {
				fmt.Printf("/login persisted PwIdCombo db=%s bucket=%s key=%s\n",
					dbHashedPwName, dbHashedPwBucket, cookieValue)
			}
			//fmt.Printf("/login pwIdCombo stored time=%v\n", time.Since(startRequestTime))
		}
	}

	readConfigLock.RLock()
	myMaxRingSecs := maxRingSecs
	myMaxTalkSecsIfNoP2p := maxTalkSecsIfNoP2p
	readConfigLock.RUnlock()
	var myHubMutex sync.RWMutex
	hub := newHub(myMaxRingSecs, myMaxTalkSecsIfNoP2p, dbEntry.StartTime)
	//fmt.Printf("/login newHub urlID=%s duration %d/%d rt=%v\n",
	//	urlID, maxRingSecs, maxTalkSecsIfNoP2p, time.Since(startRequestTime))

	exitFunc := func(calleeClient *WsClient, comment string) {
		// exitFunc: callee is logging out: release hub and port of this session

		// verify if the old calleeClient.hub.WsClientID is really same as the new wsClientID
		var reqWsClientID uint64 = 0
		if(calleeClient!=nil && calleeClient.hub!=nil) {
			reqWsClientID = calleeClient.hub.WsClientID
		}
		if reqWsClientID != wsClientID {
			// not the same: deny deletion
			fmt.Printf("exithub callee=%s abort wsID=%d/%d %s rip=%s\n",
				globalID, wsClientID, reqWsClientID, comment, remoteAddrWithPort)
			return;
		}

		fmt.Printf("exithub callee=%s wsID=%d %s %s\n",
			globalID, wsClientID, comment, remoteAddrWithPort)

		myHubMutex.Lock()
		if hub != nil {
			if globalID != "" {
				_,lenGlobalHubMap = DeleteFromHubMap(globalID)
			}
			hub = nil
		}
		myHubMutex.Unlock()

/*
		// mark wsClientMap[wsClientID] for removal
		wsClientMutex.Lock()
		wsClientData,ok := wsClientMap[wsClientID]
		if(ok) {
			wsClientData.removeFlag = true
			wsClientMap[wsClientID] = wsClientData
		}
		wsClientMutex.Unlock()
		// start wsClientMap[wsClientID] removal thread
		go func() {
			time.Sleep(60 * time.Second)
			wsClientMutex.Lock()
			wsClientData,ok := wsClientMap[wsClientID]
			if !ok {
				wsClientMutex.Unlock()
				fmt.Printf("exithub callee=%s wsClientMap[wsID=%d] fail rip=%s\n",
					globalID, wsClientID, remoteAddrWithPort)

			} else if wsClientData.removeFlag {
				//fmt.Printf("exithub callee=%s wsClientMap[wsID=%d] del rip=%s\n",
				//	globalID, wsClientID, remoteAddrWithPort)
				delete(wsClientMap, wsClientID)
				wsClientMutex.Unlock()

			} else {
				wsClientMutex.Unlock()
				var err error
                globalID,_,err = StoreCalleeInHubMap(urlID, myMultiCallees, remoteAddrWithPort, wsClientID, false)
                if err != nil {
                   fmt.Printf("exithub callee=%s wsClientMap[wsID=%d] undo err=%s rip=%s\n",
                        globalID, wsClientID, err, remoteAddrWithPort)
                } else {
                    fmt.Printf("exithub callee=%s wsClientMap[wsID=%d] undo rip=%s\n",
                        globalID, wsClientID, remoteAddrWithPort)
                }
			}
		}()
        //fmt.Printf("exithub callee=%s wsID=%d done rip=%s\n", globalID, wsClientID, remoteAddrWithPort)
*/
        wsClientMutex.Lock()
        delete(wsClientMap, wsClientID)
        wsClientMutex.Unlock()
	}

	hub.exitFunc = exitFunc
	hub.calleeUserAgent = userAgent

	wsClientMutex.Lock()
	myHubMutex.RLock()
	wsClientMap[wsClientID] = wsClientDataType{hub, dbEntry, dbUser, urlID, globalID, false}
	myHubMutex.RUnlock()
	wsClientMutex.Unlock()

	//fmt.Printf("/login newHub store in local hubMap with globalID=%s\n", globalID)
	hubMapMutex.Lock()
	myHubMutex.RLock()
	hubMap[globalID] = hub
	myHubMutex.RUnlock()
	hubMapMutex.Unlock()

	//fmt.Printf("/login run hub id=%s durationSecs=%d/%d rt=%v\n",
	//	urlID,maxRingSecs,maxTalkSecsIfNoP2p, time.Since(startRequestTime)) // rt=44ms, 113ms
	wsAddr := fmt.Sprintf("ws://%s:%d/ws", hostname, wsPort)
	readConfigLock.RLock()
	if r.TLS != nil || r.Header.Get("X-Forwarded-Proto") == "https" {
		// hand out the wss url
		if wssUrl != "" {
			wsAddr = wssUrl
		} else {
			wsAddr = fmt.Sprintf("wss://%s:%d/ws", hostname, wssPort)
		}
	} else {
		if wsUrl != "" {
			wsAddr = wsUrl
		}
	}
	readConfigLock.RUnlock()
	wsAddr = fmt.Sprintf("%s?wsid=%d", wsAddr, wsClientID)
	//if logWantedFor("wsAddr") {
	//	fmt.Printf("/login wsAddr=%s\n",wsAddr)
	//}

	fmt.Printf("/login callee=%s %v reqtime=%v rip=%s\n",
		urlID, wsClientID, time.Since(startRequestTime), remoteAddrWithPort)

	responseString := fmt.Sprintf("%s|%d|%s|%d|%v",
		wsAddr,                     // 0
		dbUser.ConnectedToPeerSecs, // 1
		outboundIP,                 // 2
		serviceSecs,                // 3
		dbUser.Int2&1 != 0)         // 4 isHiddenCallee
	fmt.Fprintf(w, responseString)

	if urlID != "" && globalID != "" {
		// start a goroutine for max X seconds to check if callee has succefully logged in via ws
		// if hub.CalleeLogin is still false then, do skv.DeleteFromHubMap(globalID)
		// to invalidate this callee/hub
		go func() {
			waitForClientWsConnectSecs := 30
			waitedFor := 0
			for i := 0; i < waitForClientWsConnectSecs; i++ {
				myHubMutex.RLock()
				if hub == nil {
					myHubMutex.RUnlock()
					break
				}
				if hub.CalleeLogin.Get() {
					myHubMutex.RUnlock()
					break
				}
				myHubMutex.RUnlock()

				time.Sleep(1 * time.Second)
				waitedFor++

				hubMapMutex.RLock()
				myHubMutex.Lock()
				hub = hubMap[globalID]
				myHubMutex.Unlock()
				hubMapMutex.RUnlock()

				myHubMutex.RLock()
				if hub == nil {
					// callee is already gone
					myHubMutex.RUnlock()
					break
				}
				myHubMutex.RUnlock()
				//if i==0 {
				//	fmt.Printf("/login checking callee id=%s for activiy in the next %ds...\n",
				//		urlID, waitForClientWsConnectSecs)
				//}
			}
			// hub.CalleeLogin will be set by callee-client sending "init|"
			myHubMutex.RLock()
			if hub != nil && !hub.CalleeLogin.Get() {
				myHubMutex.RUnlock()
				fmt.Printf("/login ws-connect timeout %ds removing %s/%s %v rip=%s\n",
					waitedFor, urlID, globalID, wsClientID, remoteAddrWithPort)
				if globalID != "" {
					//_,lenGlobalHubMap = 
						DeleteFromHubMap(globalID)
				}
				// also Unregister callee
				myHubMutex.RLock()
				if hub != nil && hub.CalleeClient != nil {
					hub.doUnregister(hub.CalleeClient, "ws-con timeout")
				}
			}
			myHubMutex.RUnlock()
		}()
	}
	return
}

func createCookie(w http.ResponseWriter, urlID string, pw string, pwIdCombo *PwIdCombo) (error,string) {
	// create new cookie with name=webcallid value=urlID
	// store only if url parameter nocookie is NOT set
	cookieSecret := fmt.Sprintf("%d", rand.Int63n(99999999999))
	if logWantedFor("cookie") {
		fmt.Printf("/login cookieSecret=%s\n", cookieSecret)
	}

	// we need urlID in cookieName only for answie#
	cookieName := "webcallid"
	if strings.HasPrefix(urlID, "answie") {
		cookieName = "webcallid-" + urlID
	}
	expiration := time.Now().Add(6 * 31 * 24 * time.Hour)
	cookieValue := fmt.Sprintf("%s&%s", urlID, string(cookieSecret))
	if logWantedFor("cookie") {
		fmt.Printf("/login create cookie cookieName=(%s) cookieValue=(%s)\n",
			cookieName, cookieValue)
	}
	cookieObj := http.Cookie{Name: cookieName, Value: cookieValue,
		Path:     "/",
		HttpOnly: false,
		SameSite: http.SameSiteStrictMode,
		Expires:  expiration}
	cookie := &cookieObj
	http.SetCookie(w, cookie)
	if logWantedFor("cookie") {
		fmt.Printf("/login cookie (%v) created\n", cookieValue)
	}

	pwIdCombo.Pw = pw
	pwIdCombo.CalleeId = urlID
	pwIdCombo.Created = time.Now().Unix()
	pwIdCombo.Expiration = expiration.Unix()

	skipConfirm := true
	return kvHashedPw.Put(dbHashedPwBucket, cookieValue, pwIdCombo, skipConfirm), cookieValue
/*
	if err != nil {
		fmt.Printf("# /login persist PwIdCombo error db=%s bucket=%s cookie=%s err=%v\n",
			dbHashedPwName, dbHashedPwBucket, cookieValue, err)
		if globalID != "" {
			_,lenGlobalHubMap = DeleteFromHubMap(globalID)
		}
		fmt.Fprintf(w, "noservice")
		return
	}

	if logWantedFor("cookie") {
		fmt.Printf("/login persisted PwIdCombo db=%s bucket=%s key=%s\n",
			dbHashedPwName, dbHashedPwBucket, cookieValue)
	}
*/
	//fmt.Printf("/login pwIdCombo stored time=%v\n",
	//	time.Since(startRequestTime))
}

