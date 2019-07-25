package example.micronaut

import groovy.transform.CompileStatic
import io.micronaut.http.MediaType
import io.micronaut.http.annotation.Controller
import io.micronaut.http.annotation.Get
import io.micronaut.http.annotation.Produces

@CompileStatic
@Controller("/")
class HelloController {
    @Produces(MediaType.TEXT_PLAIN)
    @Get("/")
    String index() {
        "Hello World"
    }
}
