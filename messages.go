package main

type Message struct {
	Payload any
}
type MessageStoreData struct {
	Key  string
	Size int64
	ID   string
}
type MessageReadData struct {
	Key string
	ID  string
}

type MessageDeleteData struct {
	Key string
	ID  string
}

type MessageSyncData struct {
	ID string
}
