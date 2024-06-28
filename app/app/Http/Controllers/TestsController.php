<?php

namespace App\Http\Controllers;

use App\Enums\TestResultEnum;
use App\Enums\TestTypeEnum;

class TestsController extends Controller
{
    public function index()
    {
        return view('tests.index', [
            'tests' => [
                (object) [
                    'id' => 1,
                    'title' => 'Login screen renders',
                    'lastResult' => TestResultEnum::notRun->value,
                    'timestamp' => now(),
                    'type' => TestTypeEnum::manual->value,
                    'error' => null,
                ],
                (object) [
                    'id' => 2,
                    'title' => 'Login screen renders',
                    'lastResult' => TestResultEnum::pass->value,
                    'timestamp' => now(),
                    'type' => TestTypeEnum::automated->value,
                    'error' => null,
                ],
                (object) [
                    'id' => 3,
                    'title' => 'Login screen renders',
                    'lastResult' => TestResultEnum::fail->value,
                    'timestamp' => now(),
                    'type' => TestTypeEnum::automated->value,
                    'error' => null,
                ],
            ],
        ]);
    }

    public function show()
    {
        return view('tests.show', [
            'test' => (object) [
                'id' => 3,
                'title' => 'Login screen renders',
                'lastResult' => TestResultEnum::fail->value,
                'timestamp' => now(),
                'type' => TestTypeEnum::automated->value,
                'error' => null,
            ],
            'runs' => [],
        ]);
    }
}
