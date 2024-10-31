<?php

namespace App\Services;

use App\Data\InstallationData;
use Illuminate\Support\Facades\Cache;

use Carbon\Carbon;
use Illuminate\Validation\UnauthorizedException;

class Installation
{
    static public function get(): ?InstallationData
    {
        $data = Cache::get('installation');

        if (!$data) {
            throw new UnauthorizedException('Installation not found.');
        }

        return  $data ? InstallationData::from($data): null;
    }

    static public function exists(): bool
    {
        return Cache::has('installation');
    }

    static public function set()
    {
        $data = GithubClient::getAccessToken(env("GITHUB_APP_INSTALATION_ID"), GithubClient::generateJWTWebToken());
        $cacheDuration = now()->diffInSeconds(Carbon::parse($data['expires_at']));
        Cache::put(
            key: 'installation',
            value: InstallationData::from([
                'access_token' => $data['token'],
                'expires_at' => $data['expires_at'],
                'repository_selection' => $data['repository_selection'],
            ])->toArray(),
            ttl: $cacheDuration
        );
    }

    static public function install(): bool
    {
        if (self::exists()) {
            $installation = self::get();
            if (now()->gte($installation->expires_at)) {
                self::set();
            }
        } else {
            self::set();
        }

        return self::exists();

    }
}
