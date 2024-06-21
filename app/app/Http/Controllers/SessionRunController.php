<?php

namespace App\Http\Controllers;

use App\Actions\Github\DispatchSessionRun;
use App\Models\Session;
use App\Models\SessionRun;
use Illuminate\Http\Request;

class SessionRunController extends Controller
{
    public function store(Session $session, DispatchSessionRun $dispatchSessionRun)
    {
        $dispatchSessionRun->handle($session);

        return redirect()->route('sessions.show', $session);
    }

    public function show(Session $session, SessionRun $run) {
        return view('sessions.run', [
            'session' => $session,
            'run' => $run,
        ]);
    }
}
