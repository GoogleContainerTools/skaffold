package hello;

import org.springframework.web.bind.annotation.RestController;
import org.springframework.web.bind.annotation.RequestMapping;

import java.nio.charset.StandardCharsets;
import java.nio.file.Files;
import java.nio.file.Paths;
import java.util.List;

@RestController
public class HelloController {

    @RequestMapping("/")
    public String index() throws Exception {
        return "text-to-replace\n";
    }
}
