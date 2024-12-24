const express = require('express')
const { echo } = require('./utils');
const app = express()
const port = 3000

app.get('/', (req, res) => res.send(echo('Hello World!')))

app.listen(port, () => console.log(`Example app listening on port ${port}!`))
