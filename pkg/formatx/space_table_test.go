package formatx

import (
	"testing"

	"github.com/MakeNowJust/heredoc"
	"github.com/stretchr/testify/assert"
)

func TestTableSpaceSeparated(t *testing.T) {
	// given
	input := heredoc.Doc(`
		NAME       	NAMESPACE  	REVISION	UPDATED                                	STATUS  	CHART                	APP VERSION
		psql       	default    	1       	2023-04-27 19:30:48.042056 +0200 CEST  	deployed	postgresql-12.2.7    	15.2.0     
		traefik    	kube-system	1       	2023-04-19 20:58:57.709052559 +0000 UTC	deployed	traefik-10.19.300    	2.6.2      
		traefik-crd	kube-system	1       	2023-04-19 20:58:56.564578223 +0000 UTC	deployed	traefik-crd-10.19.300`)

	expectedTable := Table{
		Headers: []string{"NAME", "NAMESPACE", "REVISION", "UPDATED", "STATUS", "CHART", "APP VERSION"},
		Rows: [][]string{
			{"psql", "default", "1", "2023-04-27 19:30:48.042056 +0200 CEST", "deployed", "postgresql-12.2.7", "15.2.0"},
			{"traefik", "kube-system", "1", "2023-04-19 20:58:57.709052559 +0000 UTC", "deployed", "traefik-10.19.300", "2.6.2"},
			{"traefik-crd", "kube-system", "1", "2023-04-19 20:58:56.564578223 +0000 UTC", "deployed", "traefik-crd-10.19.300", ""},
		},
	}

	expectedLines := []string{
		replaceTabsWithSpaces("NAME       	NAMESPACE  	REVISION	UPDATED                                	STATUS  	CHART                	APP VERSION"),
		replaceTabsWithSpaces("psql       	default    	1       	2023-04-27 19:30:48.042056 +0200 CEST  	deployed	postgresql-12.2.7    	15.2.0     "),
		replaceTabsWithSpaces("traefik    	kube-system	1       	2023-04-19 20:58:57.709052559 +0000 UTC	deployed	traefik-10.19.300    	2.6.2      "),
		replaceTabsWithSpaces("traefik-crd	kube-system	1       	2023-04-19 20:58:56.564578223 +0000 UTC	deployed	traefik-crd-10.19.300"),
	}

	parserTable := &TableSpace{}

	// when
	actual := parserTable.TableSeparated(input)

	// then
	assert.Equal(t, expectedTable, actual.Table)
	assert.Equal(t, expectedLines, actual.Lines)
}
