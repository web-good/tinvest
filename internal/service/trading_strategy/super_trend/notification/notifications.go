package notification

import "strings"

func Trade(shares []string) string {
	notifyMessageBuilder := strings.Builder{}
	notifyMessageBuilder.WriteString("🟢 \n<u><b>К ПОКУПКЕ:</b></u>\n\n\n<code>")

	for _, share := range shares {
		notifyMessageBuilder.WriteString("<u>" + share + "</u>\n")
	}

	notifyMessageBuilder.WriteString("</code>")

	return notifyMessageBuilder.String()
}
