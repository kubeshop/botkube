package bot

import (
	"time"

	"github.com/kubeshop/botkube/pkg/api"
)

func FixNonInteractiveSingleSection() api.Message {
	return api.Message{
		Type:      api.NonInteractiveSingleSection,
		Timestamp: time.Date(2022, 04, 21, 2, 43, 0, 0, time.UTC),
		Sections: []api.Section{
			{
				Base: api.Base{
					Header: "Section Header",
				},
				TextFields: api.TextFields{
					{
						Key:   "Kind",
						Value: "pod",
					},
					{
						Key:   "Namespace",
						Value: "botkube",
					},
					{
						Key:   "Name",
						Value: "webapp-server-68c5c57f6f",
					},
					{
						Key:   "Type",
						Value: "BackOff",
					},
				},
				BulletLists: api.BulletLists{
					{
						Title: "Messages",
						Items: []string{
							"Lorem ipsum dolor sit amet, consectetur adipiscing elit, sed do eiusmod tempor incididunt",
							"Sed ut perspiciatis unde omnis iste natus error sit voluptatem accusantium doloremque laudantium",
							"At vero eos et accusamus et iusto odio dignissimos ducimus qui blanditiis praesentium",
						},
					},
					{
						Title: "Issues",
						Items: []string{
							"Issue item 1",
							"Issue item 2",
							"Issue item 3",
						},
					},
				},
			},
		},
	}
}
