package hello;

import com.sun.net.httpserver.HttpServer;
import java.io.*;
import java.net.InetSocketAddress;

public class Application {
  public static void main(String[] args) throws IOException {
    HttpServer server = HttpServer.create(new InetSocketAddress(8080), 0);
    server.createContext("/", exchange -> {
      byte[] response = "Hello, World!".getBytes();
      exchange.sendResponseHeaders(200, response.length);
      try (OutputStream os = exchange.getResponseBody()) {
        os.write(response);
      }
    });
    server.start();
  }
}
