"""Example Python web server."""

from http.server import BaseHTTPRequestHandler, HTTPServer


class Handler(BaseHTTPRequestHandler):

    def do_GET(self):
        self.send_response(200)
        self.send_header('Content-type', 'text/plain')
        self.wfile.write(b'Hello world!\n')


def main():
    address = ('', 8000)
    server = HTTPServer(address, Handler)
    print('Serving on localhost:8000')
    server.serve_forever()


if __name__ == '__main__':
    main()