package main

import (
	"bufio"
	"fmt"
	"log"
	"net"
	"net/textproto"
	"regexp"
	"strings"
)

type Message struct {
	nick     string
	user     string
	hostname string
	target   string
	content  string
}

type Notice struct {
	from    string
	target  string
	content string
}

type PONG struct {
	content string
}

type Bot struct {
	server  string
	port    string
	nick    string
	user    string
	channel string
	pass    string
	conn    net.Conn
}

// Take a line and pack it into a struct representing a message
func ParseLine(line string) {
	re_privmsg, _ := regexp.Compile(`\:([\w\-\|\[\]\(\)]+)\!([\w\-\|\[\]\(\)])@([\w\.\-]+) PRIVMSG ([\w\-\|\[\]\(\)\#]+) \:(.+)$`)
	privmsg := re_privmsg.FindAllString(line, 5)

	re_notice, _ := regexp.Compile(`\:(.?+) NOTICE ([\w\-\|\[\]\(\)\#]+) :(.+)$`)
	notice := re_notice.FindAllString(line, 3)

	re_ping, _ := regexp.Compile(`PING :(.*)$`)
	ping := re_ping.FindAllString(line, 1)

	switch {
	default:
		return nil
	case ping != nil:
		return &PONG{
			content: ping[0]}
	case privmsg != nil:
		return &Message{
			nick:     privmsg[0],
			user:     privmsg[1],
			hostname: privmsg[2],
			target:   privmsg[3],
			content:  privmsg[4]}
	case notice != nil:
		return &Notice{
			from:    notice[0],
			target:  notice[1],
			content: notice[2]}
	}
}

func NewBot() *Bot {
	return &Bot{
		server:  "irc.freenode.net",
		port:    "6667",
		nick:    "nanagonanashuu",
		channel: "#7l7wtest",
		pass:    "",
		conn:    nil,
		user:    "nanagonanashuu"}
}

// Connect bot to IRC server
func (bot *Bot) Connect() (conn net.Conn, err error) {
	conn, err = net.Dial("tcp", bot.server+":"+bot.port)
	if err != nil {
		log.Fatal("Unable to connect to IRC server", err)
	}
	bot.conn = conn
	log.Printf("Connected to IRC server %s (%s)\n", bot.server, bot.conn.RemoteAddr())
	return bot.conn, nil
}

// Set nick, join channel
func join(conn net.Conn, ircbot *Bot) {
	fmt.Fprintf(conn, "USER %s 8 * :%s\r\n", ircbot.nick, ircbot.nick)
	fmt.Fprintf(conn, "NICK %s\r\n", ircbot.nick)
	fmt.Fprintf(conn, "JOIN %s\r\n", ircbot.channel)
}

func privmsg(conn net.Conn, target string, message string) {
	fmt.Fprintf(conn, "PRIVMSG %s %s\r\n", target, message)
}

// Check if line is PING
func ping(line string) bool {
	return strings.HasPrefix(line, "PING")
}

// Respond to PING with PONG + the random string
func pingResponse(conn net.Conn, pong *PONG) {
	fmt.Fprintf(conn, "PONG %s\r\n", pong.content)
}

func mirror(conn net.Conn, msg *Message) {
	privmsg(conn, msg.target, msg.content)
}

func main() {
	ircbot := NewBot()
	conn, _ := ircbot.Connect()
	join(conn, ircbot)
	defer conn.Close()

	reader := bufio.NewReader(conn)
	tp := textproto.NewReader(reader)
	for {
		line, err := tp.ReadLine()
		if err != nil {
			log.Fatal("Error reading connection stream")
			break
		}
		fmt.Printf("%s\n", line)
		// Pack message into struct
		msg := ParseLine(line)

		// Perform actions depending on the content of the message
		switch msg := msg.(type) {
		default:
			break
		case *PONG:
			pingResponse(conn, msg)
		case *Message:
			mirror(conn, msg)
		case *Notice:
			break
		}

	}
}
