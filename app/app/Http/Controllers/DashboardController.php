<?php

namespace App\Http\Controllers;

use App\Models\Session;
use Illuminate\Database\Eloquent\Relations\HasMany;

class DashboardController extends Controller
{
    public function index()
    {
        $sessions = Session::query()
            ->with([
                'runs' => function (HasMany $query) {
                    $query->orderBy('id', 'desc')->take(1);
                },
            ])
            ->orderBy('created_at', 'desc')
            ->take(3)
            ->get();

        return view('dashboard.index', [
            'sessions' => $sessions,
        ]);
    }
}
