package main

import ("fmt"
		"net"
		"log")
func main () {
	//criando tcp socket 
	conexao, err := net.Dial("tcp", "localhost:8080")
	if err !=nil {
		log.Fatalln(err)
	
	}
	//mandando msg
	fmt.Fprintf(conexao, "Oi server")

	//resposta do servidor
	buf := make ([]byte, 1024)
	n,err := conexao.Read(buf) //bloqueia ate retornar o numero de bytes lidos 
	
	if err != nil {
		log.Print(err)
	}
	fmt.Println(string(buf[:n]))
    conexao.Close()	

}