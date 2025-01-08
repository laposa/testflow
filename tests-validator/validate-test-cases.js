const validateSchema = require('.');
const yamlFiles = require('./load-yaml-files');

const tcdSchema = {
    name: { type: String, required: true },
    suite: { type: String, required: true },
    description: { type: String, required: true },
    steps: [
        {
            action: [{ type: String, required: true }],
            input: { required: false },
            result: [{ type: String, required: true }]
        }
    ],
}
let schemaErrors = [];

yamlFiles.forEach(yaml => {
    console.log(`checking ${yaml}`);
    const errors = validateSchema(yaml, { schema: tcdSchema });
    if(errors && errors.length) schemaErrors.push({ file: yaml, errors: errors});
});

process.on('exit', () => {
    if(schemaErrors && schemaErrors.length) console.log(`List of errors:`);
    schemaErrors.forEach(error => console.log(`\nFile: ${error.file}\n   ${JSON.stringify(error.errors)}`));
    console.log(`\ncheck finished, files checked: ${yamlFiles.length} number of errors: ${schemaErrors.length}`)
});
process.exit(schemaErrors.length)
