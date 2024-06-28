<?php

namespace App\Http\Controllers;

class MaintenanceController extends Controller
{
    public function index()
    {
        return view('maintenance.index');
    }
}
