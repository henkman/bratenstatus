package main

import (
	"fmt"
	"html/template"
	"net"
	"net/http"
	"sort"
	"sync"
	"time"

	"github.com/sauerbraten/extinfo"
)

type Info struct {
	server     *extinfo.Server
	write      sync.Mutex
	basic      extinfo.BasicInfo
	client     map[int]extinfo.ClientInfo
	freshUntil time.Time
}

type Client struct {
	Name      string
	ID        int
	IP        net.IP
	Frags     int
	Deaths    int
	Teamkills int
	Accuracy  int
	Health    int
	Weapon    string
}

type Clients []Client

func (a Clients) Len() int           { return len(a) }
func (a Clients) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a Clients) Less(i, j int) bool { return a[i].Frags > a[j].Frags }

func (info *Info) refresh() error {
	info.write.Lock()
	defer info.write.Unlock()
	basicInfo, err := info.server.GetBasicInfo()
	if err != nil {
		return err
	}
	info.basic = basicInfo
	clientInfo, err := info.server.GetAllClientInfo()
	if err != nil {
		return err
	}
	info.client = clientInfo
	info.freshUntil = time.Now().Add(time.Second * 1)
	return nil
}

func main() {
	var info Info
	{
		addr, err := net.ResolveUDPAddr("udp", "localhost:28785")
		if err != nil {
			panic(err)
		}
		server, err := extinfo.NewServer(*addr, time.Second*1)
		if err != nil {
			panic(err)
		}
		info.server = server
		info.freshUntil = time.Now()
	}
	tmpls, err := template.ParseFiles("index.html")
	if err != nil {
		panic(err)
	}
	http.ListenAndServe(":80", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if time.Now().After(info.freshUntil) {
			err := info.refresh()
			if err != nil {
				fmt.Println(err)
				http.Error(w, "error contacting server", http.StatusInternalServerError)
				return
			}
		}

		clients := make([]Client, 0, len(info.client))
		for _, client := range info.client {
			clients = append(clients, Client{
				Name:      client.Name,
				ID:        client.ClientNum,
				IP:        client.IP,
				Frags:     client.Frags,
				Deaths:    client.Deaths,
				Teamkills: client.Teamkills,
				Accuracy:  client.Accuracy,
				Health:    client.Health,
				Weapon:    client.Weapon,
			})
		}
		sort.Sort(Clients(clients))
		tmpls.Execute(w, struct {
			Description string
			Mode        string
			Map         string
			MaxClients  int
			SecsLeft    int
			Clients     []Client
		}{
			Description: info.basic.Description,
			Mode:        info.basic.GameMode,
			Map:         info.basic.Map,
			MaxClients:  info.basic.MaxNumberOfClients,
			SecsLeft:    info.basic.SecsLeft,
			Clients:     clients,
		})
	}))
}
