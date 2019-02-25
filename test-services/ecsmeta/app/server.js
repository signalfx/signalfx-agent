const PORT = 80
const DEFAULT_TARGET_IP = '127.0.0.1'

const fs = require('fs')

const express = require('express')
const app = express()

app.get('/:filename', (req, res) => {
    const filename = req.params.filename + '.json'
    let contents = fs.readFileSync(filename, 'utf8')

    // Pre-parse operations
    if(filename.indexOf('metadata') >= 0) {
        contents = contents.replace('TARGET_IP', DEFAULT_TARGET_IP)
    }

    const parsed = JSON.parse(contents)

    if(filename.indexOf('metadata') >= 0) {
        for(let i = 0; i < parsed['Containers'].length; i++) {
            if(req.query['mask_' + parsed['Containers'][i]['Name']]) {
                parsed['Containers'][i]['DockerId'] = 'masked!' // Lose the docker id to simulate missing container in the metadata]
            }
            if(req.query[parsed['Containers'][i]['Name'] + '_ip']) {
                parsed['Containers'][i]['Networks'][0]['IPv4Addresses'][0] = req.query[parsed['Containers'][i]['Name'] + '_ip']
            }
        }
    }

    res.status(200).json(parsed)
})

app.get('/:filename/:key', (req, res) => {
    const filename = req.params.filename
    const keyname = req.params.key
    const contents = fs.readFileSync(filename + '.json')
    const parsed = JSON.parse(contents)

    if(filename.indexOf('metadata') >= 0) {
        const containers = parsed['Containers']

        for(let i = 0; i < containers.length; i++) {
            if(containers[i]['DockerId'] === keyname) {
                res.status(200).json(containers[i])
                break
            }
        }
        res.status(404).send('Key not found')
    }
    else if(filename.indexOf('stats') >= 0) {
        if(parsed[keyname]) {
            res.status(200).json(parsed[keyname])
        }
        else {
            res.status(404).send('Key not found')
        }
    }
})

app.listen(PORT, () => {
    console.info(`ECS metadata endpoint is running on ${PORT}`)
})
