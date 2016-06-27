package res

type Members struct {
	Members []*Member `json:"members"`
}

type Member struct {
	IP        string `json:"ip"`
	Status    string `json:"status"`
	Namespace string `json:"namespace"`
}
