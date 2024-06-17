<?php

namespace App\Models;

use Illuminate\Database\Eloquent\Factories\HasFactory;
use Illuminate\Database\Eloquent\Model;
use Illuminate\Database\Eloquent\Relations\HasMany;

class Installation extends Model
{
    use HasFactory;

    protected $fillable = [
        'installation_id',
        'access_token',
        'repository_selection',
        'expires_at'
    ];

    public function sessions(): HasMany
    {
        return $this->hasMany(Session::class);
    }
}
