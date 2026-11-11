// Dev-only: lets http://localhost:8091 talk to a real restick server whose
// RESTICK_PUBLIC_BASE_URL is https://localhost (the API rejects any other Origin).
// Chrome/Firefox accept the Secure session cookie on localhost; Safari does not.
import http from "node:http";
const [, , listen = "8091", target = "8090"] = process.argv;
http.createServer((req, res) => {
  const headers = { ...req.headers, host: "localhost" };
  if (req.method !== "GET" && req.method !== "HEAD") headers.origin = "https://localhost";
  const up = http.request({ host: "127.0.0.1", port: Number(target), path: req.url, method: req.method, headers }, (r) => {
    res.writeHead(r.statusCode ?? 502, r.headers);
    r.pipe(res);
  });
  up.on("error", () => { res.writeHead(502); res.end("upstream down"); });
  req.pipe(up);
}).listen(Number(listen), () => console.log(`proxy :${listen} -> :${target}`));
