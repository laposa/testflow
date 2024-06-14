<?php

namespace App\Http\Controllers;

use App\Actions\Github\ExchangeGithubInstallCode;
use App\Actions\Github\FetchAvailableWorkflows;
use App\Actions\Github\InstallGithubApp;
use App\Models\Installation;
use Firebase\JWT\JWT;
use Illuminate\Support\Facades\Storage;
use Laravel\Socialite\Facades\Socialite;

class GithubAppController extends Controller
{
    public function callback(InstallGithubApp $installGithubApp)
    {

        $installGithubApp->handle( request('installation_id'), request('setup_action'));

        return redirect()->route('dashboard');
    }

}
