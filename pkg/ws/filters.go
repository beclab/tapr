package ws

import (
	"bytetrade.io/web3os/tapr/pkg/utils"
	"k8s.io/apimachinery/pkg/util/sets"
)

// If the data parameter for filtering is empty, there is no need to call the filtering function,
// otherwise it will clear the current result set.

type fields struct {
	user   string
	connId string
	client *Client
}

type Filter struct {
	data []fields
}

func NewFilter(server *Server) *Filter {

	server.RLock()
	defer server.RUnlock()

	f := &Filter{}

	for userName, userConns := range server.users {
		for connId, conn := range userConns.conns {
			f.data = append(f.data, fields{
				user:   userName,
				connId: connId,
				client: conn,
			})
		}
	}

	return f
}

func (f *Filter) Result() []string {

	r := make(sets.Set[string])
	for _, d := range f.data {
		r.Insert(d.connId)
	}

	return r.UnsortedList()
}

func (f *Filter) filter(list []string, fieldValue func(field *fields) string) *Filter {
	res := []fields{}
	if len(list) == 0 {
		f.data = res
		return f
	}

	for _, d := range f.data {
		if utils.ListContains[string](list, fieldValue(&d)) {
			res = append(res, d)
		}
	}

	f.data = res
	return f
}

func (f *Filter) FilterByUsers(users []string) *Filter {
	return f.filter(users, func(field *fields) string { return field.user })
}

func (f *Filter) FilterByConnIds(connIds []string) *Filter {
	return f.filter(connIds, func(field *fields) string { return field.connId })
}

func (f *Filter) FilterByExpired() *Filter {
	res := []fields{}
	for _, d := range f.data {
		if d.client.isExpired() {
			res = append(res, d)
		}
	}

	f.data = res

	return f
}
