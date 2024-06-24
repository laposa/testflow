<?php

namespace App\Http\Controllers;

use App\Actions\Github\FetchSessionWorkflowRuns;
use App\Actions\Github\FetchTestSuites;
use App\Actions\Session\CreateSession;
use App\Models\Installation;
use App\Models\Session;
use Illuminate\Http\Request;
use Illuminate\Support\Facades\Cache;

class SessionController extends Controller
{
    public function index() {
        return view('sessions.index', [
            'sessions' => Session::all(),
        ]);
    }

    public function show(Session $session) {
        return view('sessions.show', [
            'session' => $session
        ]);
    }

    public function create(FetchTestSuites $fetchTestSuites) {
        $suites = Cache::remember('suites', now()->addHour(), fn () => $fetchTestSuites->handle());

        $suitesWithoutWorkflow = collect($suites)->filter(fn ($suite) => !isset($suite[0]['workflow_id']))->toArray();
        $suites = collect($suites)->filter(fn ($suite) => isset($suite[0]['workflow_id']))->toArray();

        return view('sessions.create', [
            'suites' => $suites,
            'suitesWithoutWorkflow' => $suitesWithoutWorkflow,
        ]);
    }

    public function store(Request $request, CreateSession $createSession) {
        $session = $createSession->handle(Installation::first(), $request->all());

        return redirect()->route('sessions.show', $session);
    }

}
