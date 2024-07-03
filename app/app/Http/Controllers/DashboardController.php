<?php

namespace App\Http\Controllers;

use App\Models\Session;

class DashboardController extends Controller
{
    public function index()
    {
        //here?
        return view('dashboard.index', [
            'sessions' => Session::all()->reverse()->take(3),
        ]);
    }
}
