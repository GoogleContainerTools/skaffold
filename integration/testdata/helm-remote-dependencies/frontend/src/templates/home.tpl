{{ define "home" }}
<!DOCTYPE html>
<html>
<head>
    <meta charset="utf-8">
    <title>Guestbook</title>
    <meta name="viewport" content="width=device-width, initial-scale=1">
    <link rel="stylesheet" href="https://stackpath.bootstrapcdn.com/bootstrap/4.3.1/css/bootstrap.min.css" integrity="sha384-ggOyR0iXCbMQv3Xipma34MD+dH/1fQ784/j6cY/iJTQUOhcWr7x9JvoRxT2MZw1T" crossorigin="anonymous">
    <link rel="stylesheet" href="/static/style.css" type="text/css">
    <link rel="icon" href="/static/favicon.png" type="image/x-icon"/>
    <link href="https://fonts.googleapis.com/css?family=Roboto" rel="stylesheet">
</head>

<body>
    <div class="header">
        <div class="container">
            <h1>
                <a href="/">
                    My Guestbook
                </a>
            </h1>
            <a href="#" class="text-muted">View on GitHub</a>
        </div>
    </div>

    <div class="container posts mt-0">
        <form class="form-inline" method="POST" action="/post">
            <label class="sr-only" for="name">Name</label>
            <div class="input-group mb-2 mr-sm-2">
                <div class="input-group-prepend">
                    <div class="input-group-text">Your Name</div>
                </div>
                <input type="text" class="form-control" id="name" name="name" required>
            </div>
            <label class="sr-only" for="message">Message</label>
            <div class="input-group mb-2 mr-sm-2">
                <div class="input-group-prepend">
                    <div class="input-group-text">Message</div>
                </div>
                <input type="text" class="form-control" id="message" name="message" required>
            </div>
            <button type="submit" class="btn btn-primary mb-2">Post to Guestbook</button>
        </form>

        {{ range .messages }}
        <div class="card my-3 col-12">
            <div class="card-body">
                <h5 class="card-title">{{.Author}}</h5>
                <h6 class="card-subtitle mb-2 text-muted">{{ since .Date}} ago</h6>
                <br>
                <p class="card-text">
                    {{ .Message }}
                </p>
            </div>
        </div>
        {{ else }}
            <div class="alert alert-info" role="alert">
                No messages are logged to the guestbook yet.
            </div>
        {{ end }}
    </div>
</body>
</html>
{{ end }}
