<?php

namespace App\Http\Controllers;

use App\Models\Session;
use Illuminate\Database\Eloquent\Relations\BelongsToMany;

class DashboardController extends Controller
{
    public function index()
    {
        $sessions = Session::query()->with('items')->orderBy('created_at', 'desc')->take(3)->get();

        foreach ($sessions as $session) {

            $session->isRunning = false;
            $session->passed = 0;
            $session->failed = 0;

            foreach ($session->itemsGroupedByService as $service) {
                $lastRun = $service->get(0)->runs->first();
                if ($lastRun && $lastRun->result_log) {
                    $session->passed += $lastRun->parsedResults->getTotalPassed();
                    $session->failed += $lastRun->parsedResults->getTotalFailures();
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
