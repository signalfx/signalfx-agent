const http = require('http')
const port = 3000

const numMetrics = parseInt(process.env.NUM_METRICS || '100', 10);
const numExtraDimensions = Math.min(parseInt(process.env.NUM_EXTRA_DIMENSIONS || '1', 10), 26);

function makePrometheusMetrics() {
	var out = 
		"# HELP http_requests_total The total number of HTTP requests.\n" +
	    "# TYPE http_requests_total counter\n";

	for (var i = 0; i < numMetrics; i++) {
		out += `sample_metric{index="${i}"${extraLabels()}} 1027\n`
	}
	return out;
}

function extraLabels() {
	var labelStr = "";
	for (var i = 0; i < numExtraDimensions; i++) {
		labelStr += `,${String.fromCharCode(97+i)}="${i}"`;
	}
	return labelStr;
}

const requestHandler = (request, response) => {
  response.end(makePrometheusMetrics());
}

const server = http.createServer(requestHandler)

server.listen(port, (err) => {
  if (err) {
    return console.log('something bad happened', err)
  }

  console.log(`server is listening on ${port}`)
})
