package main

import (
	"bufio"
	"errors"
	"fmt"
	"log"
	"net"
	"net/textproto"
	"regexp"
	"strings"
)

type Message struct {
	from    string
	command string
	to      string
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
func ParseLine(line string) (*Message, error) {
	re, err := regexp.Compile(`(.?*) ?([A-Z]+) ([\w\-\|\[\]\(\)\#\*]*) ?:(.+)$`)
	log.Printf("%s", err)
	msg := re.FindAllString(line, 4)
	if msg != nil {
		return &Message{
			from:    msg[0],
			command: msg[1],
			to:      msg[2],
			content: msg[3]}, nil
	} else {
		return &Message{}, errors.New(fmt.Sprintf("No command in line %s", line))
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
func pingResponse(conn net.Conn, pong *Message) {
	fmt.Fprintf(conn, "PONG %s\r\n", pong.content)
}

func mirror(conn net.Conn, msg *Message) {
	privmsg(conn, msg.to, msg.content)
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
		msg, err := ParseLine(line)
		// Perform actions depending on the content of the message
		if err == nil {
			switch {
			case msg.command == "PONG":
				pingResponse(conn, msg)
			case msg.command == "PRIVMSG":
				mirror(conn, msg)
			}
		}

	}
}
