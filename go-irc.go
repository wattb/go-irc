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
func PackMessage(line string) *Message {
	re, _ := regexp.Compile(`\:([\w\-\|\[\]\(\)]+)\!([\w\-\|\[\]\(\)])@([\w\.\-]+) PRIVMSG ([\w\-\|\[\]\(\)\#]+) \:(.+)$`)
	match := re.FindAllString(line, 5)

	return &Message{
		nick:     match[0],
		user:     match[1],
		hostname: match[2],
		target:   match[3],
		content:  match[4]}
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
func pingResponse(line string, conn net.Conn) {
	pong := strings.Split(line, "PING")[1]
	fmt.Fprintf(conn, "PONG %s\r\n", pong)
}

func atSelf(msg *Message, ircbot *Bot) bool {
	return (msg.target == ircbot.nick)
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
		msg := PackMessage(line)

		// Perform actions depending on the content of the message
		switch {
		case ping(line):
			pingResponse(line, conn)
		case atSelf(msg, ircbot):
			mirror(conn, msg)
		}

	}
}
