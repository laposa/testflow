<?php

namespace App\Http\Controllers;

use App\Models\Session;
use App\Models\SessionRun;

class SessionRunController extends Controller
{
    public function show(Session $session, SessionRun $run) {
        return view('sessions.run', [
            'session' => $session,
            'run' => $run,
        ]);
    }
}
