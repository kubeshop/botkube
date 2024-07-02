package config

import (
	"fmt"
)

const redactedSecretStr = "*** REDACTED ***"

// HideSensitiveInfo removes sensitive information from the config.
func HideSensitiveInfo(in Config) Config {
	out := in
	// TODO: avoid printing sensitive data without need to resetting them manually (which is an error-prone approach)
	for key, val := range out.Communications {
		val.SocketSlack.AppToken = redactedSecretStr
		val.SocketSlack.BotToken = redactedSecretStr
		val.Elasticsearch.Password = redactedSecretStr
		val.Discord.Token = redactedSecretStr
		val.Mattermost.Token = redactedSecretStr
		val.CloudSlack.Token = redactedSecretStr
		// To keep the printed config readable, we don't print the certificate bytes.
		val.CloudSlack.Server.TLS.CACertificate = nil
		val.CloudTeams.Server.TLS.CACertificate = nil

		// Replace private channel names with aliases
		cloudSlackChannels := make(IdentifiableMap[CloudSlackChannel])
		for _, channel := range val.CloudSlack.Channels {
			if channel.Alias == nil {
				cloudSlackChannels[channel.ChannelBindingsByName.Name] = channel
				continue
			}

			outChannel := channel
			outChannel.ChannelBindingsByName.Name = fmt.Sprintf("%s (public alias)", *channel.Alias)
			outChannel.Alias = nil
			cloudSlackChannels[*channel.Alias] = outChannel
		}
		val.CloudSlack.Channels = cloudSlackChannels

		// maps are not addressable: https://stackoverflow.com/questions/42605337/cannot-assign-to-struct-field-in-a-map
		out.Communications[key] = val
	}

	return out
}
