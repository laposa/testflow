<?php

namespace App\Http\Controllers;

use App\Models\Session;
use App\Models\SessionServiceRun;
use App\Models\SessionServiceSuiteTest;
use App\Services\GithubClient;

class SessionRunController extends Controller
{
    public function show(Session $session, SessionServiceRun $run)
    {
        $run->load('service.suites.tests');

        return view('sessions.run', [
            'session' => $session,
            'run' => $run,
        ]);
    }

    public function showTestFile(
        Session $session,
        SessionServiceRun $run,
        SessionServiceSuiteTest $test,
    ) {
        $client = new GithubClient();

        $fileContent = $client->fetchRepositoryContents(
            $run->service->repository_name,
            $test->path,
            $run->commit_sha,
        );

        $content = base64_decode($fileContent['content']);

        return response($content)->header('Content-Type', 'text/plain');
    }
}
