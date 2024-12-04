const express = require('express');
const bodyParser = require('body-parser');
const fs = require('fs');
const yaml = require('js-yaml');

const app = express();
const port = 3000;

app.use(bodyParser.urlencoded({ extended: true }));
app.set('view engine', 'ejs');
app.use(express.static('public'));

app.get('/', (req, res) => {
    res.render('form');
});

app.post('/generate-yaml', (req, res) => {
    const {
        testName,
        suite,
        description,
        action,
        input,
        result
    } = req.body;

    const steps = action.map((act, index) => {
        if(input) {
            return {
                action: act.split('\r\n'),
                input: { ... input.reduce((acc, inpt) => {
                    acc[inpt.name] = { description: inpt.description, value: inpt.value };
                    return acc;
                }, {})},
                result: result[index].split('\r\n')
            }
        } 
        return {
            action: act.split('\r\n'),
            result: result[index].split('\r\n')
        }

    });

    const yamlData = {
        name: testName,
        suite: suite,
        description: description,
        steps: steps
    };

    const yamlString = yaml.dump(yamlData);

    // Save YAML to file
    if (!fs.existsSync('./output')) {
        fs.mkdir('./output', (err) => {
            if (err) {
                console.error('Error creating directory:', err);
            };
        });
    }
    const filePath = `./output/${testName || 'test'}.yaml`;
    fs.writeFileSync(filePath, yamlString);

    res.send(`YAML file generated: <a href="/output/${testName || 'test'}.yaml" download>Download YAML</a>`);
});

app.use('/output', express.static('output'));

app.listen(port, () => {
    console.log(`Server running at http://localhost:${port}`);
});
