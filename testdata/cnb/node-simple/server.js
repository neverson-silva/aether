const http = require("http");
http.createServer((req, res) => {
  res.writeHead(200, { "Content-Type": "text/plain" });
  res.end("aether node fixture\n");
}).listen(process.env.PORT || 8080);
