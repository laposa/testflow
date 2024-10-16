<?php

namespace App\Http\Controllers;

use App\Models\Session;
use Illuminate\Http\Request;

class SessionReportController extends Controller
{
    //

    public function executive(Session $session)
    {
        $session->load('runs.service.suites.tests');
        return view('sessions.report', [
            'session' => $session,
            'showLog' => false
        ]);
    }

    public function full(Session $session)
    {
        $session->load('runs.service.suites.tests');
        return view('sessions.report', [
            'session' => $session,
            'showLog' => true
        ]);
    }
}
