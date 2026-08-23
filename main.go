package main

// FONTE: #80 Golang - Master TCP & UDP Sockets – Complete Guide
import (
	"fmt"
		"log"
		"net"
	"time")

func gerenciarConexaoNaoBloqueante (conexao net.Conn){
	defer conexao.Close()

	for{
		conexao.SetDeadline(time.Now().Add(time.Second))

		buf:= make([]byte, 1024)
		n, err := conexao.Read(buf)
		
		if err != nil {
			if netErr, ok := err.(net.Error); ok && netErr.Timeout(){
				continue
			} else {
				log.Println("Conexao fechada", err)
				break
			}
		}

		fmt.Println("Recebido", string(buf[:n]))
		conexao.SetWriteDeadline(time.Now().Add(time.Second))

		fmt.Fprintf(conexao, "OPAAA NO TEMPO CORRETO")
		if err != nil {
			log.Println ("Eror de escrita dsoi cliente", err)
			break
		}
	}
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
		go gerenciarConexaoNaoBloqueante(novaConexao)
	}

}
