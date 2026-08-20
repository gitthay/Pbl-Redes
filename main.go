package main

// FONTE: #80 Golang - Master TCP & UDP Sockets – Complete Guide
import (
	"fmt"
		"log"
		"net")

func gerenciarConexao (conexao net.Conn) {
	defer conexao.Close() 

	// Ler do clinete 
	buf := make ([]byte, 1024)
	n,err := conexao.Read(buf) //bloqueia ate retornar o numero de bytes lidos 
	
	if err != nil {
		log.Print(err)
	}
	fmt.Println(string(buf[:n]))
	fmt.Fprintf(conexao, "Opa" + string (buf[:n]))

	}

func main(){
	//Listener aguarda o cliente cria um canal (socket) e encaminha para uma rotina
	ln, err := net.Listen("tcp", "localhost:8080") //protocolo e porta a qual será executada
	if err != nil {
		log.Fatalln(err)
	}
	defer ln.Close() //agendo o fechamento de ln quando main for encerrada

	for {
		//Esperando conexao
		novaConexao, err := ln.Accept()
		if err != nil {
			log.Println(err)
		}
		go gerenciarConexao(novaConexao)
	}

}
