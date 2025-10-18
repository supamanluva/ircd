package commands

// IRC numeric replies as defined in RFC 1459 and RFC 2812
const (
	// Welcome messages
	RPL_WELCOME          = "001"
	RPL_YOURHOST         = "002"
	RPL_CREATED          = "003"
	RPL_MYINFO           = "004"
	RPL_ISUPPORT         = "005"

	// Command responses
	RPL_UMODEIS          = "221"
	RPL_AWAY             = "301"
	RPL_USERHOST         = "302"
	RPL_ISON             = "303"
	RPL_UNAWAY           = "305"
	RPL_NOWAWAY          = "306"
	RPL_WHOISUSER        = "311"
	RPL_WHOISSERVER      = "312"
	RPL_WHOISOPERATOR    = "313"
	RPL_ENDOFWHO         = "315"
	RPL_WHOISIDLE        = "317"
	RPL_ENDOFWHOIS       = "318"
	RPL_WHOISCHANNELS    = "319"
	RPL_LISTSTART        = "321"
	RPL_LIST             = "322"
	RPL_LISTEND          = "323"
	RPL_CHANNELMODEIS    = "324"
	RPL_NOTOPIC          = "331"
	RPL_TOPIC            = "332"
	RPL_INVITING         = "341"
	RPL_WHOREPLY         = "352"
	RPL_NAMREPLY         = "353"
	RPL_ENDOFNAMES       = "366"
	RPL_MOTD             = "372"
	RPL_MOTDSTART        = "375"
	RPL_ENDOFMOTD        = "376"
	RPL_YOUREOPER        = "381"

	// Error messages
	ERR_NOSUCHNICK       = "401"
	ERR_NOSUCHSERVER     = "402"
	ERR_NOSUCHCHANNEL    = "403"
	ERR_CANNOTSENDTOCHAN = "404"
	ERR_TOOMANYCHANNELS  = "405"
	ERR_NORECIPIENT      = "411"
	ERR_NOTEXTTOSEND     = "412"
	ERR_UNKNOWNCOMMAND   = "421"
	ERR_NONICKNAMEGIVEN  = "431"
	ERR_ERRONEUSNICKNAME = "432"
	ERR_NICKNAMEINUSE    = "433"
	ERR_USERNOTINCHANNEL = "441"
	ERR_NOTONCHANNEL     = "442"
	ERR_USERONCHANNEL    = "443"
	ERR_NOTREGISTERED    = "451"
	ERR_NEEDMOREPARAMS   = "461"
	ERR_ALREADYREGISTERED = "462"
	ERR_PASSWDMISMATCH   = "464"
	ERR_CHANNELISFULL    = "471"
	ERR_UNKNOWNMODE      = "472"
	ERR_BADCHANNELKEY    = "475"
	ERR_NOPRIVILEGES     = "481"
	ERR_CHANOPRIVSNEEDED = "482"
	ERR_UMODEUNKNOWNFLAG = "501"
	ERR_USERSDONTMATCH   = "502"
)

// NumericReply formats a numeric reply message
func NumericReply(serverName, code, nick, message string) string {
	if nick == "" {
		nick = "*"
	}
	return ":" + serverName + " " + code + " " + nick + " " + message
}
