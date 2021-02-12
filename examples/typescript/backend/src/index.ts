import express, { Response } from 'express';
import { echo } from './utils';
const app = express()
const port = 3000

app.get('/', (_, res: Response) => res.send(echo('Hello World!')))

app.listen(port, () => console.log(`Example app listening on port ${port}!`))
