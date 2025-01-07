# Test Case Validator

This project provides a test case validator that can be run using npm. The validator checks the test cases in the specified directories to ensure they meet the required standards.

## Getting Started

### Prerequisites

- Node.js
- npm

### Installation

1. Install the dependencies:
    ```sh
    npm install
    ```

## Usage

To run the test case validator, use the following command:

```sh
npm run validate -- [directory1] [directory2] ...
```

Replace `[directory1]`, `[directory2]`, etc., with the root paths of the directories you want to validate.

## Config

You can specify the paths to yaml files along with the extension in `config.js` file

```js
module.exports = {
    // List of directories to search for test cases
    testRootDirectories: ['./tests/mobile', './tests/api', './tests/web'],
    // Extension of the test case files
    testCaseExtension: '.manual.yaml',
}

```

### Example

```sh
npm run validate -- ./tests/mobile ./tests/api ./tests/web
```

This command will validate the test cases in the `./tests/mobile`, `./tests/api` and `./tests/web` directories.

## License

This project is licensed under the MIT License.
