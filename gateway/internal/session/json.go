package session

// JSON 序列化时 nil slice 会变成 null，前端需要 []。
func CoalesceConversations(list []*Conversation) []*Conversation {
	if list == nil {
		return []*Conversation{}
	}
	return list
}

func CoalesceMessages(list []Message) []Message {
	if list == nil {
		return []Message{}
	}
	return list
}
