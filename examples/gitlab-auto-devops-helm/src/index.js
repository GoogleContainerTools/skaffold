const express = require('express')
const { echo } = require('./utils')

const app = express()
const port = process.env.PORT || 3000

app.get('/', (_, res) => res.send(echo('Hello World!')))
app.get('/version', (_, res) => res.json(process.env.npm_package_version))
app.get('/:username', (req, res) => {
    res.send(echo(`Hello, ${req.params.username}!`))
})

app.listen(port, () => console.log(`Example app listening on port ${port}!`))
