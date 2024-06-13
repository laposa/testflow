<?php

namespace App\Http\Controllers;

use App\Enums\TestResultEnum;
use App\Enums\TestTypeEnum;
use Illuminate\Http\Request;

class SessionsController extends Controller
{
    public function index() {
        $sessions = [
            (object) [
                "id" => 1,
                "title" => "Session 1",
                "timestamp" => now(),
                "issuer" => "Issuer 1"
            ]
        ];

        return view('sessions.index', [
            'sessions' => $sessions
        ]);
    }

    public function show() {
        return view('sessions.show', [
            'session' =>  (object) [
                "id" => 1,
                "title" => "Session 1",
                "timestamp" => now(),
                "issuer" => "Issuer 1",
                "runs" => [
                    (object) [
                        "id" => 1,
                        "timestamp" => now(),
                        "result" => TestResultEnum::pass->value,
                        "suite" => "Suite 1",
                        "session" => "Session 1",
                        "code" => "",
                        "error" => "",
                        'test' => (object) [
                            'title' => 'Login screen renders',
                        ]
                    ]
                ],
            ],
            'suites' => [
                (object) [
                    'id' => 1,
                    'title' => 'Suite 1',
                    'passed' => 4,
                    'failed' => 2,
                    'tests' => [
                        (object) [
                            'id' => 1,
                            'title' => 'Login screen renders',
                            'lastResult' => TestResultEnum::notRun->value,
                            'timestamp'=> now(),
                            'type' => TestTypeEnum::manual->value,
                            'error' => null
                        ],
                        (object) [
                            'id' => 2,
                            'title' => 'Login screen renders',
                            'lastResult' => TestResultEnum::pass->value,
                            'timestamp'=> now(),
                            'type' => TestTypeEnum::automated->value,
                            'error' => null
                        ],
                    ]
                ]
            ]
        ]);
    }

    public function create() {
        return view('sessions.create', [
            'suites' => [
                (object) [
                    'id' => 1,
                    'title' => 'Suite 1',
                    'passed' => 4,
                    'failed' => 2,
                    'tests' => [
                        (object) [
                            'id' => 1,
                            'title' => 'Login screen renders',
                            'lastResult' => TestResultEnum::notRun->value,
                            'timestamp'=> now(),
                            'type' => TestTypeEnum::manual->value,
                            'error' => null
                        ],
                        (object) [
                            'id' => 2,
                            'title' => 'Login screen renders',
                            'lastResult' => TestResultEnum::pass->value,
                            'timestamp'=> now(),
                            'type' => TestTypeEnum::automated->value,
                            'error' => null
                        ],
                    ]
                ]
            ]
        ]);
    }

    public function store() {
        return view('sessions.show');
    }

}
