<?php

use Illuminate\Database\Migrations\Migration;
use Illuminate\Database\Schema\Blueprint;
use Illuminate\Support\Facades\Schema;

return new class extends Migration
{
    /**
     * Run the migrations.
     */
    public function up(): void
    {
        Schema::table('test_sessions', function (Blueprint $table) {
            $table->dropForeign(['installation_id']);
            $table->dropColumn('installation_id');
        });
        
        Schema::dropIfExists('installations');
    }

    /**
     * Reverse the migrations.
     */
    public function down(): void
    {
        //
    }
};
