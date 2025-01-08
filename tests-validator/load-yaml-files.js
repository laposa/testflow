const fs = require('fs');
const path = require('path');
const config = require('./config');
const paths = process.argv.slice(2);

// Find all yaml files in the given directory having the given extension defined in config.js
function findManualYamlFiles(rootDirectory) {
  let results = [];
  if(!fs.existsSync(rootDirectory)) {
    return results;
  }
  const files = fs.readdirSync(rootDirectory);
  files.forEach(file => {
    const fullPath = path.join(rootDirectory, file);
    if (fs.statSync(fullPath).isDirectory()) {
      results = results.concat(findManualYamlFiles(fullPath));
    } else if (file.endsWith(config.testCaseExtension)) {
      results.push(fullPath);
    }
  });
  return results;
}
const yamlFiles = config.testRootDirectories.concat(paths).map(rootDir => findManualYamlFiles(rootDir)).flat();
module.exports = yamlFiles;

