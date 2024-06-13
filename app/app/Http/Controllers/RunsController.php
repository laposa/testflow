<?php

namespace App\Http\Controllers;

use App\Enums\TestResultEnum;
use Illuminate\Http\Request;

class RunsController extends Controller
{
    public function index()
    {
        return view('runs.index', [
            'runs' => [
                (object) [
                    "id" => 1,
                    "timestamp" => now(),
                    "result" => TestResultEnum::pass->value,
                    "suite" => "Suite 1",
                    "session" => "Session 1",
                    "code" => "",
                    "error" => "",
                    "test" => (object) [
                        "id" => 1,
                        "title" => "Login screen renders",
                        "lastResult" => TestResultEnum::pass->value,
                        "timestamp" => now(),
                        "type" => "manual",
                        "error" => null
                    ]
                ]
            ]
        ]);
    }
}
