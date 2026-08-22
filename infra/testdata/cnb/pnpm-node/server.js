const http=require("http");http.createServer((q,s)=>{s.writeHead(200);s.end("aether pnpm fixture\n")}).listen(process.env.PORT||8080);
