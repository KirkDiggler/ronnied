package main

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
)

func main() {
	fmt.Println("Starting Ronnied - Discord Dice Drinking Game Bot")
	
	// TODO: Initialize configuration
	// TODO: Setup Discord connection
	// TODO: Register command handlers
	// TODO: Initialize services and repositories
	
	// Keep the bot running until interrupted
	fmt.Println("Bot is now running. Press CTRL-C to exit.")
	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt)
	<-sc
	
	// Cleanup before exit
	fmt.Println("Shutting down...")
}
