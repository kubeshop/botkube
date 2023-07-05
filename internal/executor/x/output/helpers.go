package output

import "github.com/kubeshop/botkube/pkg/api"

func noItemsMsg() api.Message {
	return api.Message{
		Sections: []api.Section{
			{
				Base: api.Base{
					Description: "Not found.",
				},
			},
		},
	}
}
