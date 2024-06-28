<?php

namespace App\Actions\Github;

use App\Models\Installation;
use Illuminate\Support\Facades\Http;

class ExchangeGithubInstallCode
{
    public function handle(string $code, string $installationId, string $action)
    {
        //        $access_token = $this->generateInstallationAccessToken($installationId);
        //
        //        Installation::updateOrCreate([
        //            'installation_id' => $installationId
        //        ], [
        //            'access_token' => $access_token,
        //            'repository_selection' => 'all'
        //        ]);
    }

    //    protected function generateUserAccessToken(string $code)
    //    {
    //        $response = Http::withHeaders([
    //            'Accept' => 'application/json'
    //        ])->post('https://github.com/login/oauth/access_token', [
    //            'client_id' => env('GITHUB_CLIENT_ID'),
    //            'client_secret' => env('GITHUB_CLIENT_SECRET'),
    //            'code' => $code,
    //        ]);
    //
    //        $data = $response->json();
    //
    //        dd($data);
    //
    //        return $data['access_token'];
    //    }
    //
}
