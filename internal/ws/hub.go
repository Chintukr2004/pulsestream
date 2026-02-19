package ws


import(
	"log"
	"sync"
	"github.com/Chintukr2004/pulsestream/internal/model"
)
type Hub struct {
	clients map[*Client]bool
	broadcast chan model.Post

	mu sync.Mutex
}

func NewHub() *Hub{
	return &Hub{
		clients: make(map[*Client]bool),
		broadcast: make(chan model.Post),

	}
}

func (h *Hub) Run(){
	for post := range h.broadcast{
		h.mu.Lock()
		for client:= range h.clients{
			err:= client.conn.WriteJSON(post)
			if err != nil{
				log.Println("Write error:", err)
				client.conn.Close()
				delete(h.clients, client)
			}
		}
		h.mu.Unlock()
	}
}

func ( h*Hub)Register ( client *Client){
	h.mu.Lock()
	h.clients[client] = true
	h.mu.Unlock()
}

func (h *Hub)Unregistor ( client *Client){
	h.mu.Lock()
	delete(h.clients, client)
	h.mu.Unlock()
}

func (h *Hub) Broadcast(post model.Post) {
	h.broadcast <- post
}
