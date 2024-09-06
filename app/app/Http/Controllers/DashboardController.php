<?php

namespace App\Http\Controllers;

use App\Models\Session;

class DashboardController extends Controller
{
    public function index()
    {
        $sessions = Session::query()
            ->with('lastRuns')
            ->orderBy('created_at', 'desc')
            ->take(3)
            ->get();

        foreach ($sessions as $session) {
            $session->isRunning = false;
            $session->passed = 0;
            $session->failed = 0;

            foreach ($session->lastRuns as $run) {
                if ($run->passed > 0 || $run->failed > 0) {
                    $session->passed += $run->passed;
                    $session->failed += $run->failed;
                } else {
                    $session->isRunning = true;
                }
            }
        }

        return view('dashboard.index', [
            'sessions' => $sessions,
        ]);
    }
}
