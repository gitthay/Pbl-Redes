package main

// FONTE: #80 Golang - Master TCP & UDP Sockets – Complete Guide
import (
	"fmt"
		"log"
		"net"
	"time")

func gerenciarConexaoNaoBloqueante (conexao net.Conn) {
	defer conexao.Close() 

	for{
		//deadline para ler
		conexao.SetReadDeadline(time.Now().Add(time.Second))
		// Ler do clinete 
		buf := make ([]byte, 1024)
		n,err := conexao.Read(buf) //bloqueia ate retornar o numero de bytes lidos 
		
		if err != nil {
			if netErr, ok := err.(net.Error); ok && netErr.Timeout(){
				continue
			} else {
				fmt.Println("Conexao Fechada: ", err)
				break
			}
		}
		//sucesso
		fmt.Println("Recebida: ", string (buf[:n]))

		//deadline
		conexao.SetWriteDeadline(time.Now().Add(time.Second))

		//mesnagem para o cliente 
		fmt.Fprintln(conexao, "Ouvi!")

		if err != nil{
			log.Println("Erro para escrever para cliente", err)
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
