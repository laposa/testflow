<?php

namespace App\Enums;

enum TestResultEnum: string
{
    case notRun = 'notRun';
    case pass = 'pass';
    case fail = 'fail';
    case skipped = 'skipped';
    case error = 'error';
}
