<?php

namespace App\Http\Controllers;

use App\Actions\Github\DispatchSessionRun;
use App\Models\Session;
use Illuminate\Http\Request;

class SessionRunController extends Controller
{
    public function store(Request $request, Session $session, DispatchSessionRun $dispatchSessionRun)
    {
        $dispatchSessionRun->handle($session, $request->all());

        return redirect()->route('sessions.show', $session);
    }

    public function show(Session $session) {
        $session = $session->append('workflow_runs');
        ray($session->workflow_runs);
        return view('sessions.run', [
            'session' => $session
        ]);
    }
}
