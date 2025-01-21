<?php

namespace App\Http\Controllers;

use App\Actions\Session\CreateSession;
use App\Models\Session;
use Illuminate\Http\Request;

class SessionController extends Controller
{
    public function index()
    {
        $sessions = Session::query()
            ->with(['lastRuns', 'issuer'])
            ->orderBy('id', 'desc')
            ->get();

        foreach ($sessions as $session) {
            $session->isRunning = false;
            $session->passed = 0;
            $session->failed = 0;
            $session->skipped = 0;

            foreach ($session->lastRuns as $run) {
                if ($run->passed > 0 || $run->failed > 0 || $run->status == 'success') {
                    $session->passed += $run->passed;
                    $session->failed += $run->failed;
                    $session->skipped += $run->skipped;
                } else {
                    $session->isRunning = true;
                }
            }
        }

        return view('sessions.index', [
            'sessions' => $sessions,
        ]);
    }

    public function show(Session $session)
    {
        $session = $session->load('services.suites.tests');
        $reviewRequests = $session->reviewRequests()->where('status', 'pending')->get();

        return view('sessions.show', [
            'session' => $session,
            'reviewRequests' => $reviewRequests,
        ]);
    }

    public function create()
    {
        return view('sessions.create');
    }

    public function store(Request $request, CreateSession $createSession)
    {
        $session = $createSession->handle($request->all());

        return redirect()->route('sessions.show', $session);
    }
}
