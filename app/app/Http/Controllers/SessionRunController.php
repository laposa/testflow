<?php

namespace App\Http\Controllers;

use App\Models\Session;
use App\Models\SessionServiceRun;

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
}
