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
    if (!fs.existsSync('./output')) {
        fs.mkdir('./output', (err) => {
            if (err) {
                console.error('Error creating directory:', err);
            };
        });
    }
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
        const filteredInputs = input ? input.filter(inp => Number.parseInt(inp.relatedStep) === index) : [];
        if(filteredInputs.length > 0) {
            return {
                action: act.split('\r\n'),
                input: { ... filteredInputs.reduce((acc, inpt) => {
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

    const filePath = `./output/${testName || 'test'}.yaml`;
    fs.writeFileSync(filePath, yamlString);

    res.send(`YAML file generated: <a href="/output/${testName || 'test'}.yaml" download>Download YAML</a>`);
});

app.use('/output', express.static('output'));

app.listen(port, () => {
    console.log(`Server running at http://localhost:${port}`);
});
